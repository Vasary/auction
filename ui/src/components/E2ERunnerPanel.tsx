import { useRef, useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./Card"
import { Button } from "./Button"
import { Input } from "./Input"

type E2EAuctionResult = {
  tenderId: string
  startAt: string
  endAt: string
  expectedWinnerId: string | null
  expectedCurrentPrice: number | null
  bidsAttempted: number
  bidsAccepted: number
  bidsPersisted: number
  winnerId: string | null
  currentPrice: number
  status: string
  wsConnected: number
  passed: boolean
  reasons: string[]
}

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

export default function E2ERunnerPanel() {
  const [e2eConfig, setE2eConfig] = useState({
    auctionsCount: 4,
    clientsPerAuction: 4,
    startDelaySec: 60,
    shortDurationSec: 60,
    longDurationSec: 120,
    startPrice: 10000,
    step: 100,
    bidRounds: 3,
    bidIntervalMs: 220,
  })
  const [e2eRunning, setE2eRunning] = useState(false)
  const [e2eLogs, setE2eLogs] = useState<string[]>([])
  const [e2eResults, setE2eResults] = useState<E2EAuctionResult[]>([])
  const [e2eSummary, setE2eSummary] = useState<any>(null)
  const e2eAbortRef = useRef(false)
  const e2eSocketsRef = useRef<WebSocket[]>([])

  const appendE2ELog = (message: string, payload?: unknown) => {
    const line = `[${new Date().toISOString()}] ${message}${payload !== undefined ? ` ${JSON.stringify(payload)}` : ""}`
    setE2eLogs((prev) => [...prev, line])
  }

  const closeAllE2ESockets = () => {
    for (const socket of e2eSocketsRef.current) {
      try {
        socket.close()
      } catch {
        // no-op
      }
    }
    e2eSocketsRef.current = []
  }

  const stopE2ERun = () => {
    e2eAbortRef.current = true
    closeAllE2ESockets()
    appendE2ELog("E2E stop requested by user")
    setE2eRunning(false)
  }

  const runE2E = async () => {
    if (e2eRunning) return

    e2eAbortRef.current = false
    setE2eRunning(true)
    setE2eLogs([])
    setE2eResults([])
    setE2eSummary(null)

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
    const wsHost = `${protocol}//${window.location.host}`
    const request = async (path: string, init?: RequestInit) => {
      const res = await fetch(path, init)
      const text = await res.text()
      let body: any = text
      try {
        body = text ? JSON.parse(text) : null
      } catch {
        // keep text
      }
      return { res, body }
    }

    try {
      appendE2ELog("E2E run started", e2eConfig)
      const runSingleAuction = async (auctionIndex: number): Promise<E2EAuctionResult> => {
        if (e2eAbortRef.current) throw new Error("aborted")

        const tenderId = crypto.randomUUID()
        const createdBy = crypto.randomUUID()
        const startOffsetSec = auctionIndex % 2 === 0 ? 0 : e2eConfig.startDelaySec
        const durationSec = auctionIndex % 2 === 0 ? e2eConfig.shortDurationSec : e2eConfig.longDurationSec
        const startAt = new Date(Date.now() + startOffsetSec * 1000)
        const endAt = new Date(startAt.getTime() + durationSec * 1000)
        const startAtUTC = startAt.toISOString()
        const endAtUTC = endAt.toISOString()

        appendE2ELog("Creating auction", { tenderId, startAtUTC, endAtUTC })
        const createResp = await request("/auctions", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            tenderId,
            startPrice: e2eConfig.startPrice,
            step: e2eConfig.step,
            startAt: startAtUTC,
            endAt: endAtUTC,
            createdBy,
          }),
        })

        if (createResp.res.status !== 201) {
          appendE2ELog("Create failed", { tenderId, status: createResp.res.status, body: createResp.body })
          return {
            tenderId,
            startAt: startAtUTC,
            endAt: endAtUTC,
            expectedWinnerId: null,
            expectedCurrentPrice: null,
            bidsAttempted: 0,
            bidsAccepted: 0,
            bidsPersisted: 0,
            winnerId: null,
            currentPrice: 0,
            status: "create_failed",
            wsConnected: 0,
            passed: false,
            reasons: [`create failed: HTTP ${createResp.res.status}`],
          }
        }

        const clients = Array.from({ length: e2eConfig.clientsPerAuction }).map(() => ({
          companyId: crypto.randomUUID(),
          personId: crypto.randomUUID(),
        }))

        for (const client of clients) {
          const pResp = await request(`/auctions/${encodeURIComponent(tenderId)}/participate`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ companyId: client.companyId }),
          })
          if (pResp.res.status >= 300) {
            appendE2ELog("Participant registration failed", {
              tenderId,
              companyId: client.companyId,
              status: pResp.res.status,
              body: pResp.body,
            })
          }
        }

        let wsConnected = 0
        let finishedEvents = 0
        let bidsAttempted = 0
        let bidsAccepted = 0
        let expectedWinnerId: string | null = null
        let expectedCurrentPrice: number | null = null
        const wsClients: Array<{ ws: WebSocket; companyId: string; personId: string }> = []

        await Promise.all(
          clients.map((client) => {
            return new Promise<void>((resolve) => {
              const ws = new WebSocket(`${wsHost}/ws/${encodeURIComponent(tenderId)}`)
              e2eSocketsRef.current.push(ws)
              wsClients.push({ ws, companyId: client.companyId, personId: client.personId })
              const timeout = setTimeout(() => resolve(), 5000)

              ws.onopen = () => {
                wsConnected++
                clearTimeout(timeout)
                resolve()
              }
              ws.onerror = () => {
                clearTimeout(timeout)
                resolve()
              }
              ws.onmessage = (event) => {
                try {
                  const data = JSON.parse(event.data)
                  if (data.type === "finished" || data.type === "auction_finished") {
                    finishedEvents++
                  }
                  if (typeof data.accepted === "boolean" && data.accepted) {
                    bidsAccepted++
                    const msgPrice = typeof data.currentPrice === "number" ? data.currentPrice : null
                    const msgWinner = typeof data.winnerId === "string" ? data.winnerId : client.companyId
                    if (
                      msgPrice !== null &&
                      (expectedCurrentPrice === null || msgPrice < expectedCurrentPrice)
                    ) {
                      expectedCurrentPrice = msgPrice
                      expectedWinnerId = msgWinner
                    }
                  }
                } catch {
                  // ignore malformed messages in e2e observer
                }
              }
            })
          })
        )

        const msUntilStart = startAt.getTime() - Date.now() + 1200
        if (msUntilStart > 0) {
          appendE2ELog("Waiting for auction start", { tenderId, waitMs: msUntilStart })
          await sleep(msUntilStart)
        }

        let bidValue = e2eConfig.startPrice
        for (let round = 0; round < e2eConfig.bidRounds; round++) {
          for (let i = 0; i < wsClients.length; i++) {
            if (e2eAbortRef.current) throw new Error("aborted")
            bidValue -= e2eConfig.step
            if (bidValue < 1) {
              break
            }

            const currentClient = wsClients[i]
            if (currentClient && currentClient.ws.readyState === WebSocket.OPEN) {
              bidsAttempted++
              currentClient.ws.send(
                JSON.stringify({
                  type: "place_bid",
                  bid: bidValue,
                  companyId: currentClient.companyId,
                  personId: currentClient.personId,
                })
              )
            }
            await sleep(e2eConfig.bidIntervalMs)
          }
        }

        const finishDeadline = endAt.getTime() + 15000
        let auctionInfo: any = null
        while (Date.now() < finishDeadline) {
          if (e2eAbortRef.current) throw new Error("aborted")
          const getResp = await request(`/auctions/${encodeURIComponent(tenderId)}`)
          if (getResp.res.status === 200) {
            auctionInfo = getResp.body
            if (auctionInfo?.status === "Finished") {
              break
            }
          }
          await sleep(1000)
        }

        const bidsResp = await request(`/auctions/${encodeURIComponent(tenderId)}/bids`)
        const bids: any[] = Array.isArray(bidsResp.body) ? bidsResp.body : []
        const reasons: string[] = []

        if (!auctionInfo || auctionInfo.status !== "Finished") {
          reasons.push("auction did not reach Finished in expected time")
        }
        if (wsConnected === 0) {
          reasons.push("no websocket clients connected")
        }
        if (wsConnected < e2eConfig.clientsPerAuction) {
          reasons.push(`connected clients less than configured: ${wsConnected}/${e2eConfig.clientsPerAuction}`)
        }
        if (bidsAttempted === 0) {
          reasons.push("no bids were attempted")
        }
        if (bidsAccepted === 0) {
          reasons.push("no bid_result with accepted=true received")
        }
        if (bidsAccepted > 0 && (expectedWinnerId === null || expectedCurrentPrice === null)) {
          reasons.push("accepted bids observed but expected winner/price were not captured")
        }
        if (bids.length === 0) {
          reasons.push("no bids persisted in backend")
        }

        for (let i = 1; i < bids.length; i++) {
          const prev = bids[i - 1]
          const cur = bids[i]
          if (!(cur.bidAmount < prev.bidAmount)) {
            reasons.push("bid amounts are not strictly descending")
            break
          }
          if ((prev.bidAmount - cur.bidAmount) % e2eConfig.step !== 0) {
            reasons.push("bid step alignment violated")
            break
          }
        }

        if (bids.length > 0) {
          const lastBid = bids[bids.length - 1]
          if (expectedWinnerId !== null && expectedWinnerId !== lastBid.companyId) {
            reasons.push("expected winner does not match last persisted bid company")
          }
          if (expectedCurrentPrice !== null && expectedCurrentPrice !== lastBid.bidAmount) {
            reasons.push("expected price does not match last persisted bid amount")
          }
          if (auctionInfo?.winnerId !== lastBid.companyId) {
            reasons.push("winnerId does not match last bid company")
          }
          if (auctionInfo?.currentPrice !== lastBid.bidAmount) {
            reasons.push("currentPrice does not match last bid amount")
          }
          if (expectedWinnerId !== null && auctionInfo?.winnerId !== expectedWinnerId) {
            reasons.push("auction winnerId does not match expected winner")
          }
          if (expectedCurrentPrice !== null && auctionInfo?.currentPrice !== expectedCurrentPrice) {
            reasons.push("auction currentPrice does not match expected price")
          }
        } else if (auctionInfo) {
          if (auctionInfo.currentPrice !== e2eConfig.startPrice) {
            reasons.push("no bids case: currentPrice differs from startPrice")
          }
        }

        const passed = reasons.length === 0
        const result: E2EAuctionResult = {
          tenderId,
          startAt: startAtUTC,
          endAt: endAtUTC,
          expectedWinnerId,
          expectedCurrentPrice,
          bidsAttempted,
          bidsAccepted,
          bidsPersisted: bids.length,
          winnerId: auctionInfo?.winnerId ?? null,
          currentPrice: auctionInfo?.currentPrice ?? 0,
          status: auctionInfo?.status ?? "unknown",
          wsConnected,
          passed,
          reasons,
        }

        appendE2ELog("Auction result", {
          tenderId,
          passed,
          wsConnected,
          expectedWinnerId,
          expectedCurrentPrice,
          bidsPersisted: bids.length,
          finishedEvents,
          reasons,
        })
        return result
      }

      const tasks = Array.from({ length: e2eConfig.auctionsCount }, (_, index) => {
        return runSingleAuction(index)
          .then((result) => {
            setE2eResults((prev) => [...prev, result])
            return result
          })
          .catch((err: any) => {
            const fallback: E2EAuctionResult = {
              tenderId: crypto.randomUUID(),
              startAt: new Date().toISOString(),
              endAt: new Date().toISOString(),
              expectedWinnerId: null,
              expectedCurrentPrice: null,
              bidsAttempted: 0,
              bidsAccepted: 0,
              bidsPersisted: 0,
              winnerId: null,
              currentPrice: 0,
              status: "run_failed",
              wsConnected: 0,
              passed: false,
              reasons: [err?.message ?? String(err)],
            }
            setE2eResults((prev) => [...prev, fallback])
            return fallback
          })
      })

      const results = await Promise.all(tasks)
      const passed = results.filter((x) => x.passed).length
      const failed = results.length - passed
      setE2eSummary({
        total: results.length,
        passed,
        failed,
      })
      appendE2ELog("E2E run completed", { total: results.length, passed, failed })
    } catch (err: any) {
      if (String(err?.message || "").includes("aborted")) {
        appendE2ELog("E2E run aborted")
      } else {
        appendE2ELog("E2E run failed", { error: err?.message ?? String(err) })
      }
    } finally {
      closeAllE2ESockets()
      setE2eRunning(false)
    }
  }

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle>Multi-auction functional scenario</CardTitle>
        <CardDescription>
          Create multiple auctions, connect multiple WS clients, place bids, wait for finish, and validate winners.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="space-y-2">
            <label className="text-xs font-medium text-muted-foreground uppercase">Auctions</label>
            <Input
              type="number"
              min={1}
              value={e2eConfig.auctionsCount}
              onChange={(e) => setE2eConfig({ ...e2eConfig, auctionsCount: Number(e.target.value) || 1 })}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-medium text-muted-foreground uppercase">Clients / Auction</label>
            <Input
              type="number"
              min={1}
              value={e2eConfig.clientsPerAuction}
              onChange={(e) => setE2eConfig({ ...e2eConfig, clientsPerAuction: Number(e.target.value) || 1 })}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-medium text-muted-foreground uppercase">Start Delay (sec)</label>
            <Input
              type="number"
              min={0}
              value={e2eConfig.startDelaySec}
              onChange={(e) => setE2eConfig({ ...e2eConfig, startDelaySec: Number(e.target.value) || 0 })}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-medium text-muted-foreground uppercase">Short Duration (sec)</label>
            <Input
              type="number"
              min={5}
              value={e2eConfig.shortDurationSec}
              onChange={(e) => setE2eConfig({ ...e2eConfig, shortDurationSec: Number(e.target.value) || 5 })}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-medium text-muted-foreground uppercase">Long Duration (sec)</label>
            <Input
              type="number"
              min={5}
              value={e2eConfig.longDurationSec}
              onChange={(e) => setE2eConfig({ ...e2eConfig, longDurationSec: Number(e.target.value) || 5 })}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-medium text-muted-foreground uppercase">Start Price</label>
            <Input
              type="number"
              min={1}
              value={e2eConfig.startPrice}
              onChange={(e) => setE2eConfig({ ...e2eConfig, startPrice: Number(e.target.value) || 1 })}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-medium text-muted-foreground uppercase">Step</label>
            <Input
              type="number"
              min={1}
              value={e2eConfig.step}
              onChange={(e) => setE2eConfig({ ...e2eConfig, step: Number(e.target.value) || 1 })}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-medium text-muted-foreground uppercase">Bid Rounds</label>
            <Input
              type="number"
              min={1}
              value={e2eConfig.bidRounds}
              onChange={(e) => setE2eConfig({ ...e2eConfig, bidRounds: Number(e.target.value) || 1 })}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-medium text-muted-foreground uppercase">Bid Interval (ms)</label>
            <Input
              type="number"
              min={100}
              value={e2eConfig.bidIntervalMs}
              onChange={(e) => setE2eConfig({ ...e2eConfig, bidIntervalMs: Number(e.target.value) || 100 })}
            />
          </div>
        </div>

        <div className="flex gap-2">
          <Button onClick={runE2E} disabled={e2eRunning}>
            {e2eRunning ? "Running..." : "Run E2E Scenario"}
          </Button>
          <Button variant="destructive" onClick={stopE2ERun} disabled={!e2eRunning}>
            Stop
          </Button>
          <Button variant="outline" onClick={() => { setE2eLogs([]); setE2eResults([]); setE2eSummary(null) }} disabled={e2eRunning}>
            Clear Results
          </Button>
        </div>

        {e2eSummary && (
          <pre className="p-3 bg-muted rounded text-xs overflow-auto">
            {JSON.stringify(e2eSummary, null, 2)}
          </pre>
        )}

        <div className="space-y-2">
          <label className="text-xs font-medium text-muted-foreground uppercase">Per-auction results</label>
          <pre className="p-4 bg-muted rounded-xl text-[11px] font-mono min-h-[100px] overflow-y-auto whitespace-pre-wrap">
            {e2eResults.length > 0
              ? JSON.stringify(e2eResults, null, 2)
              : "No results yet"}
          </pre>
        </div>

        <div className="space-y-2">
          <label className="text-xs font-medium text-muted-foreground uppercase">Execution log</label>
          <pre className="p-4 bg-muted rounded-xl text-[11px] font-mono min-h-[100px] overflow-y-auto whitespace-pre-wrap">
            {e2eLogs.length > 0 ? e2eLogs.join("\n") : "Waiting for run..."}
          </pre>
        </div>
      </CardContent>
    </Card>
  )
}

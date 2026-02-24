import { useState, useEffect, useRef } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/Card"
import { Button } from "../components/Button"
import { Input } from "../components/Input"
import { Code } from "../components/Code"
import { Activity, Plus, Clipboard, Zap, Wifi, Send, Home } from "lucide-react"
import { Link } from "react-router-dom"

export default function RequestGenerator() {
  // Health check
  const [healthOut, setHealthOut] = useState<any>(null)
  const checkHealth = async () => {
    try {
      const res = await fetch('/health')
      const text = await res.text()
      setHealthOut({ status: res.status, body: text })
    } catch (err: any) {
      setHealthOut({ status: 0, body: err.toString() })
    }
  }

  // Create Auction
  const [createForm, setCreateForm] = useState({
    tenderId: "",
    startPrice: 1000,
    step: 100,
    startAt: "",
    endAt: "",
    createdBy: ""
  })
  const [createOut, setCreateOut] = useState<any>(null)

  useEffect(() => {
    const now = new Date()
    const start = new Date(now.getTime() + 2 * 60 * 60 * 1000)
    const end = new Date(now.getTime() + 6 * 60 * 60 * 1000)
    
    const formatLocal = (d: Date) => {
      return d.toISOString().slice(0, 16)
    }

    setCreateForm(prev => ({
      ...prev,
      startAt: formatLocal(start),
      endAt: formatLocal(end)
    }))
  }, [])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const body = {
        ...createForm,
        startAt: new Date(createForm.startAt).toISOString(),
        endAt: new Date(createForm.endAt).toISOString()
      }
      const res = await fetch('/auctions', {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body)
      })
      const data = await res.json()
      setCreateOut({ status: res.status, body: data })
    } catch (err: any) {
      setCreateOut({ status: 0, body: err.toString() })
    }
  }

  // Update Auction
  const [updateForm, setUpdateForm] = useState({
    tenderId: "",
    startPrice: undefined as number | undefined,
    step: undefined as number | undefined,
    startAt: "",
    endAt: ""
  })
  const [updateOut, setUpdateOut] = useState<any>(null)

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const body: any = {}
      if (updateForm.startPrice !== undefined) body.startPrice = updateForm.startPrice
      if (updateForm.step !== undefined) body.step = updateForm.step
      if (updateForm.startAt) body.startAt = new Date(updateForm.startAt).toISOString()
      if (updateForm.endAt) body.endAt = new Date(updateForm.endAt).toISOString()

      const res = await fetch(`/auctions/${encodeURIComponent(updateForm.tenderId)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body)
      })
      const data = await res.json()
      setUpdateOut({ status: res.status, body: data })
    } catch (err: any) {
      setUpdateOut({ status: 0, body: err.toString() })
    }
  }

  // Delete Auction
  const [deleteId, setDeleteId] = useState("")
  const [deleteOut, setDeleteOut] = useState<any>(null)

  const handleDelete = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!window.confirm(`Are you sure you want to delete auction ${deleteId}?`)) return
    try {
      const res = await fetch(`/auctions/${encodeURIComponent(deleteId)}`, {
        method: "DELETE"
      })
      const text = await res.text()
      let body
      try {
        body = JSON.parse(text)
      } catch {
        body = text || "Deleted"
      }
      setDeleteOut({ status: res.status, body })
    } catch (err: any) {
      setDeleteOut({ status: 0, body: err.toString() })
    }
  }

  // Get Auction
  const [getId, setGetId] = useState("")
  const [getOut, setGetOut] = useState<any>(null)

  const handleGet = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await fetch(`/auctions/${encodeURIComponent(getId)}`)
      const data = await res.json()
      setGetOut({ status: res.status, body: data })
    } catch (err: any) {
      setGetOut({ status: 0, body: err.toString() })
    }
  }

  // List Bids
  const [listBidsId, setListBidsId] = useState("")
  const [listBidsOut, setListBidsOut] = useState<any>(null)

  const handleListBids = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await fetch(`/auctions/${encodeURIComponent(listBidsId)}/bids`)
      const data = await res.json()
      setListBidsOut({ status: res.status, body: data })
    } catch (err: any) {
      setListBidsOut({ status: 0, body: err.toString() })
    }
  }

  // Register Participant
  const [partId, setPartId] = useState({ tenderId: "", companyId: "" })
  const [partOut, setPartOut] = useState<any>(null)

  const handleParticipate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await fetch(`/auctions/${encodeURIComponent(partId.tenderId)}/participate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ companyId: partId.companyId })
      })
      const text = await res.text()
      let body
      try {
        body = JSON.parse(text)
      } catch {
        body = text || "Success"
      }
      setPartOut({ status: res.status, body })
    } catch (err: any) {
      setPartOut({ status: 0, body: err.toString() })
    }
  }

  // UUID Tools
  const [uuids, setUuids] = useState<string[]>([])
  const generateUuid = () => {
    const id = crypto.randomUUID()
    setUuids(prev => [...prev, id])
  }

  // WebSocket Playground
  const [wsConfig, setWsConfig] = useState({ tenderId: "", companyId: "", personId: "", bid: 900 })
  const [wsStatus, setWsStatus] = useState("disconnected")
  const [wsLogs, setWsLogs] = useState<string[]>([])
  const [currentWsPrice, setCurrentWsPrice] = useState<any>(null)
  const wsRef = useRef<WebSocket | null>(null)

  const appendWsLog = (msg: string, obj?: any) => {
    const line = `[${new Date().toISOString()}] ${msg} ${obj ? JSON.stringify(obj) : ""}`
    setWsLogs(prev => [...prev, line])
  }

  const connectWs = () => {
    if (!wsConfig.tenderId) return
    if (wsRef.current) wsRef.current.close()

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
    const url = `${protocol}//${window.location.host}/ws/${encodeURIComponent(wsConfig.tenderId)}`
    
    appendWsLog("INFO", `connecting to ${url}`)
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => {
      setWsStatus("connected")
      appendWsLog("OPEN", "connected")
    }
    ws.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data)
        appendWsLog("MESSAGE", data)
        const p = data.payload
        if (p.currentPrice !== undefined) {
          setCurrentWsPrice({ price: p.currentPrice, winner: p.winnerId, type: data.type })
        } else if (p.snapshot) {
          setCurrentWsPrice({ price: p.snapshot.currentPrice, winner: p.snapshot.winnerId, type: data.type })
        }
      } catch {
        appendWsLog("MESSAGE(raw)", e.data)
      }
    }
    ws.onclose = (e) => {
      setWsStatus("disconnected")
      appendWsLog("CLOSE", { code: e.code, reason: e.reason })
    }
    ws.onerror = () => appendWsLog("ERROR", "websocket error")
  }

  const sendBid = () => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return
    const payload = {
      type: "place_bid",
      bid: Number(wsConfig.bid),
      companyId: wsConfig.companyId,
      personId: wsConfig.personId
    }
    appendWsLog("SEND", payload)
    wsRef.current.send(JSON.stringify(payload))
  }

  return (
    <div className="container mx-auto px-4 py-8 space-y-8">
      <div className="flex items-center justify-between">
        <div>
           <h1 className="text-3xl font-bold">Request Generator</h1>
           <p className="text-muted-foreground">HTTP + WebSocket playground for manual testing</p>
        </div>
        <Link to="/ui">
          <Button variant="outline">
            <Home className="mr-2 h-4 w-4" />
            Home
          </Button>
        </Link>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {/* UUID Tools - First, spanning 2 blocks */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Zap className="h-5 w-5 text-amber-500" />
              UUID Tools
            </CardTitle>
            <CardDescription>Generate test identifiers</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex gap-2">
              <Button onClick={generateUuid} className="flex-1">Generate</Button>
              <Button variant="outline" onClick={() => setUuids([])}>Clear</Button>
            </div>
            <div className="space-y-2 max-h-40 overflow-y-auto">
              {uuids.map((id, i) => (
                <div key={i} className="flex items-center justify-between p-0 bg-muted rounded text-xs font-mono">
                  {id}
                  <Button size="icon" variant="ghost" className="h-6 w-6" onClick={() => navigator.clipboard.writeText(id)}>
                    <Clipboard className="h-3 w-3" />
                  </Button>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Health Check - To the right of UUID */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5 text-emerald-500" />
              Health check
            </CardTitle>
            <CardDescription>GET <Code>/health</Code></CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={checkHealth} className="w-full">Ping</Button>
            {healthOut && (
              <pre className="mt-4 p-2 bg-muted rounded text-xs overflow-auto max-h-40">
                {`HTTP ${healthOut.status}\n\n${healthOut.body}`}
              </pre>
            )}
          </CardContent>
        </Card>

        {/* Create Auction */}
        <Card className="lg:row-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Plus className="h-5 w-5 text-blue-500" />
              Create auction
            </CardTitle>
            <CardDescription>POST <Code>/auctions</Code></CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground uppercase">Tender ID</label>
              <Input placeholder="tender-123" value={createForm.tenderId} onChange={e => setCreateForm({...createForm, tenderId: e.target.value})} />
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div className="space-y-2">
                <label className="text-xs font-medium text-muted-foreground uppercase">Start Price</label>
                <Input type="number" value={createForm.startPrice} onChange={e => setCreateForm({...createForm, startPrice: Number(e.target.value)})} />
              </div>
              <div className="space-y-2">
                <label className="text-xs font-medium text-muted-foreground uppercase">Step</label>
                <Input type="number" value={createForm.step} onChange={e => setCreateForm({...createForm, step: Number(e.target.value)})} />
              </div>
            </div>
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground uppercase">Start At (Local)</label>
              <Input type="datetime-local" value={createForm.startAt} onChange={e => setCreateForm({...createForm, startAt: e.target.value})} />
            </div>
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground uppercase">End At (Local)</label>
              <Input type="datetime-local" value={createForm.endAt} onChange={e => setCreateForm({...createForm, endAt: e.target.value})} />
            </div>
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground uppercase">Created By</label>
              <Input placeholder="user-id" value={createForm.createdBy} onChange={e => setCreateForm({...createForm, createdBy: e.target.value})} />
            </div>
            <Button onClick={handleCreate} className="w-full">Create Auction</Button>
            {createOut && (
              <pre className="mt-4 p-2 bg-muted rounded text-xs overflow-auto max-h-60">
                {`HTTP ${createOut.status}\n\n${JSON.stringify(createOut.body, null, 2)}`}
              </pre>
            )}
          </CardContent>
        </Card>

        {/* Update Auction */}
        <Card className="lg:row-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5 text-orange-500" />
              Update auction
            </CardTitle>
            <CardDescription>PATCH <Code>/auctions/{"{tenderId}"}</Code></CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground uppercase">Tender ID (target)</label>
              <Input placeholder="tender-123" value={updateForm.tenderId} onChange={e => setUpdateForm({...updateForm, tenderId: e.target.value})} />
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div className="space-y-2">
                <label className="text-xs font-medium text-muted-foreground uppercase">Start Price (opt)</label>
                <Input type="number" placeholder="1000" value={updateForm.startPrice ?? ""} onChange={e => setUpdateForm({...updateForm, startPrice: e.target.value ? Number(e.target.value) : undefined})} />
              </div>
              <div className="space-y-2">
                <label className="text-xs font-medium text-muted-foreground uppercase">Step (opt)</label>
                <Input type="number" placeholder="100" value={updateForm.step ?? ""} onChange={e => setUpdateForm({...updateForm, step: e.target.value ? Number(e.target.value) : undefined})} />
              </div>
            </div>
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground uppercase">Start At (Local, opt)</label>
              <Input type="datetime-local" value={updateForm.startAt} onChange={e => setUpdateForm({...updateForm, startAt: e.target.value})} />
            </div>
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground uppercase">End At (Local, opt)</label>
              <Input type="datetime-local" value={updateForm.endAt} onChange={e => setUpdateForm({...updateForm, endAt: e.target.value})} />
            </div>
            <Button onClick={handleUpdate} className="w-full">Update Auction</Button>
            {updateOut && (
              <pre className="mt-4 p-2 bg-muted rounded text-xs overflow-auto max-h-60">
                {`HTTP ${updateOut.status}\n\n${JSON.stringify(updateOut.body, null, 2)}`}
              </pre>
            )}
          </CardContent>
        </Card>

        {/* Get Auction */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5 text-indigo-500" />
              Get auction
            </CardTitle>
            <CardDescription>GET <Code>/auctions/{"{tenderId}"}</Code></CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground uppercase">Tender ID</label>
              <Input placeholder="tender-123" value={getId} onChange={e => setGetId(e.target.value)} />
            </div>
            <Button onClick={handleGet} className="w-full">Get Auction</Button>
            {getOut && (
              <pre className="mt-4 p-2 bg-muted rounded text-xs overflow-auto max-h-40">
                {`HTTP ${getOut.status}\n\n${JSON.stringify(getOut.body, null, 2)}`}
              </pre>
            )}
          </CardContent>
        </Card>

        {/* Delete Auction */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5 text-red-500" />
              Delete auction
            </CardTitle>
            <CardDescription>DELETE <Code>/auctions/{"{tenderId}"}</Code></CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground uppercase">Tender ID</label>
              <Input placeholder="tender-123" value={deleteId} onChange={e => setDeleteId(e.target.value)} />
            </div>
            <Button onClick={handleDelete} variant="destructive" className="w-full">Delete Auction</Button>
            {deleteOut && (
              <pre className="mt-4 p-2 bg-muted rounded text-xs overflow-auto max-h-40">
                {`HTTP ${deleteOut.status}\n\n${JSON.stringify(deleteOut.body, null, 2)}`}
              </pre>
            )}
          </CardContent>
        </Card>

        {/* Register Participant */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5 text-teal-500" />
              Register participant
            </CardTitle>
            <CardDescription>POST <Code>/auctions/{"{tenderId}"}/participate</Code></CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground uppercase">Tender ID</label>
              <Input placeholder="tender-123" value={partId.tenderId} onChange={e => setPartId({...partId, tenderId: e.target.value})} />
            </div>
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground uppercase">Company ID</label>
              <Input placeholder="company-1" value={partId.companyId} onChange={e => setPartId({...partId, companyId: e.target.value})} />
            </div>
            <Button onClick={handleParticipate} className="w-full">Participate</Button>
            {partOut && (
              <pre className="mt-4 p-2 bg-muted rounded text-xs overflow-auto max-h-40">
                {`HTTP ${partOut.status}\n\n${JSON.stringify(partOut.body, null, 2)}`}
              </pre>
            )}
          </CardContent>
        </Card>

        {/* List Bids */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5 text-cyan-500" />
              List bids
            </CardTitle>
            <CardDescription>GET <Code>/auctions/{"{tenderId}"}/bids</Code></CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-xs font-medium text-muted-foreground uppercase">Tender ID</label>
              <Input placeholder="tender-123" value={listBidsId} onChange={e => setListBidsId(e.target.value)} />
            </div>
            <Button onClick={handleListBids} className="w-full">List Bids</Button>
            {listBidsOut && (
              <pre className="mt-4 p-2 bg-muted rounded text-xs overflow-auto max-h-40">
                {`HTTP ${listBidsOut.status}\n\n${JSON.stringify(listBidsOut.body, null, 2)}`}
              </pre>
            )}
          </CardContent>
        </Card>
      </div>

      <hr className="border-t-2 border-muted" />

      <div>
        <h2 className="text-xl font-bold mb-4 flex items-center gap-2">
          <Wifi className="h-5 w-5 text-purple-500" />
          WebSocket section
        </h2>
        {/* WebSocket Playground */}
        <Card className="w-full">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              WebSocket Playground
            </CardTitle>
            <CardDescription>WS <Code>ws://host/ws/{"{tenderId}"}</Code></CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              {/* Configuration and Bid */}
              <div className="lg:col-span-2 space-y-4">
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  <div className="space-y-2">
                    <label className="text-xs font-medium text-muted-foreground uppercase">Tender ID</label>
                    <Input placeholder="tender-123" value={wsConfig.tenderId} onChange={e => setWsConfig({...wsConfig, tenderId: e.target.value})} />
                  </div>
                  <div className="space-y-2">
                    <label className="text-xs font-medium text-muted-foreground uppercase">Company ID</label>
                    <Input placeholder="company-1" value={wsConfig.companyId} onChange={e => setWsConfig({...wsConfig, companyId: e.target.value})} />
                  </div>
                  <div className="space-y-2">
                    <label className="text-xs font-medium text-muted-foreground uppercase">Person ID</label>
                    <Input placeholder="person-1" value={wsConfig.personId} onChange={e => setWsConfig({...wsConfig, personId: e.target.value})} />
                  </div>
                </div>
                
                <div className="flex gap-2">
                  <Button onClick={connectWs} className="flex-1" variant={wsStatus === 'connected' ? 'outline' : 'default'}>
                    {wsStatus === 'connected' ? 'Reconnect' : 'Connect WS'}
                  </Button>
                  <Button variant="destructive" onClick={() => wsRef.current?.close()} disabled={wsStatus !== 'connected'}>
                    Disconnect
                  </Button>
                </div>

                <div className="pt-4 border-t space-y-4">
                  <div className="flex items-end gap-2">
                    <div className="flex-1 space-y-2">
                      <label className="text-xs font-medium text-muted-foreground uppercase">Bid Amount</label>
                      <Input type="number" value={wsConfig.bid} onChange={e => setWsConfig({...wsConfig, bid: Number(e.target.value)})} />
                    </div>
                    <Button onClick={sendBid} disabled={wsStatus !== 'connected'} className="px-8">
                      <Send className="mr-2 h-4 w-4" />
                      Send Bid
                    </Button>
                  </div>
                </div>
              </div>

              {/* Price Display */}
              <div className="flex flex-col justify-center">
                 <div className="p-6 bg-primary/5 border rounded-xl flex flex-col items-center justify-center h-full">
                    <div className="text-xs font-medium text-muted-foreground uppercase mb-2">Current Price</div>
                    <div className="text-5xl font-bold text-primary">{currentWsPrice?.price ?? "—"}</div>
                    {currentWsPrice && (
                      <div className="text-xs text-muted-foreground mt-4 flex flex-wrap justify-center gap-4">
                        <span>Type: <Code className="text-xs">{currentWsPrice.type}</Code></span>
                        {currentWsPrice.winner && <span>Winner: <Code className="text-xs">{currentWsPrice.winner}</Code></span>}
                      </div>
                    )}
                 </div>
              </div>
            </div>

            {/* Full-width Event Log */}
            <div className="space-y-2 pt-4 border-t">
              <div className="flex items-center justify-between">
                <label className="text-xs font-medium text-muted-foreground uppercase">Event Log</label>
                <Button variant="ghost" size="sm" onClick={() => setWsLogs([])} className="h-6 text-[10px]">Clear Log</Button>
              </div>
              <pre className="p-4 bg-muted rounded-xl text-[11px] font-mono min-h-[100px] overflow-y-auto whitespace-pre-wrap transition-all duration-300">
                 {wsLogs.map((log, i) => <div key={i} className="mb-1 border-b border-muted-foreground/10 pb-1 last:border-0">{log}</div>).reverse()}
                 {wsLogs.length === 0 && <div className="text-muted-foreground italic">Waiting for events...</div>}
              </pre>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

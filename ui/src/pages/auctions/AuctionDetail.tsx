import { useEffect, useState, useRef, useCallback } from "react"
import { useParams, useNavigate } from "react-router-dom"
import { Card, CardContent, CardHeader, CardTitle } from "../../components/Card"
import { Button } from "../../components/Button"
import { Badge } from "../../components/Badge"
import { Code } from "../../components/Code"
import { Input } from "../../components/Input"
import { ArrowLeft, Send, Building2, User, RefreshCw, Users } from "lucide-react"
import { cn, formatCurrency } from "../../lib/utils"

interface AuctionDetailProps {
  personId: string
  companyId: string
}

interface Bid {
  id: number
  companyId: string
  personId: string
  bidAmount: number
  createdAt: string
}

export default function AuctionDetail({ personId, companyId }: AuctionDetailProps) {
  const { tenderId } = useParams<{ tenderId: string }>()
  const navigate = useNavigate()
  const [wsStatus, setWsStatus] = useState<'connected' | 'disconnected' | 'connecting'>('disconnected')
  const [currentPrice, setCurrentPrice] = useState<number | null>(null)
  const [step, setStep] = useState<number>(100) // Default 100 cents
  const [isFinished, setIsFinished] = useState(false)
  const [winnerId, setWinnerId] = useState<string | null>(null)
  const [bidAmount, setBidAmount] = useState<string>("")
  const [isRefreshingBids, setIsRefreshingBids] = useState(false)
  const [lastBidTime, setLastBidTime] = useState<number>(0)
  const [cooldown, setCooldown] = useState<number>(0)
  const [logs, setLogs] = useState<{ time: string, type: string, data: any }[]>([])
  const [bids, setBids] = useState<Bid[]>([])
  const [participants, setParticipants] = useState<string[]>([])
  const wsRef = useRef<WebSocket | null>(null)
  const logContainerRef = useRef<HTMLPreElement>(null)

  const addLog = useCallback((type: string, data: any) => {
    setLogs(prev => [...prev.slice(-49), { time: new Date().toLocaleTimeString(), type, data }])
  }, [])

  const fetchAuction = useCallback(async () => {
    if (!tenderId) return
    try {
      const response = await fetch(`/auctions/${encodeURIComponent(tenderId)}`)
      if (response.ok) {
        const data = await response.json()
        setParticipants(data.participants || [])
        if (currentPrice === null) {
          setCurrentPrice(data.currentPrice)
        }
        if (winnerId === null) {
          setWinnerId(data.winnerId)
        }
        setStep(data.step)
      }
    } catch (error) {
      console.error("Failed to fetch auction:", error)
    }
  }, [tenderId, currentPrice, winnerId])

  const fetchBids = useCallback(async () => {
    if (!tenderId) return
    setIsRefreshingBids(true)
    try {
      const response = await fetch(`/auctions/${encodeURIComponent(tenderId)}/bids`)
      if (response.ok) {
        const data = await response.json()
        setBids(data)
      }
    } catch (error) {
      console.error("Failed to fetch bids:", error)
    } finally {
      setIsRefreshingBids(false)
    }
  }, [tenderId])

  useEffect(() => {
    fetchAuction()
    fetchBids()
  }, [fetchAuction, fetchBids])

  useEffect(() => {
    if (!tenderId) return

    let isUnmounted = false
    let reconnectTimeout: ReturnType<typeof setTimeout>

    const connect = () => {
      if (isUnmounted) return

      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
      const url = `${protocol}//${window.location.host}/ws/${encodeURIComponent(tenderId)}`
      
      setWsStatus('connecting')
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        setWsStatus('connected')
        addLog('INFO', 'Connected to auction')
      }

      ws.onclose = () => {
        setWsStatus('disconnected')
        addLog('INFO', 'Connection closed')
        
        if (!isUnmounted) {
          addLog('INFO', 'Reconnecting in 2 seconds...')
          reconnectTimeout = setTimeout(connect, 2000)
        }
      }

      ws.onmessage = (e) => {
        const data = JSON.parse(e.data)
        addLog('EVENT', data)
        
        if (data.type === 'auction_finished' || data.type === 'finished') {
          setIsFinished(true)
        }

        const payload = data.payload
        if (!payload) return

        if (data.type === 'snapshot') {
          setCurrentPrice(payload.currentPrice)
          setWinnerId(payload.winnerId)
          if (payload.status === 'finished' || payload.status === 'Finished') {
            setIsFinished(true)
          }
          const s = payload.step || step
          setStep(s)
          setBidAmount(((payload.currentPrice - s) / 100).toString())
          fetchBids()
        } else if (data.type === 'price_updated') {
          setCurrentPrice(payload.currentPrice)
          setWinnerId(payload.winnerId)
          const s = payload.step || step
          setStep(s)
          setBidAmount(((payload.currentPrice - s) / 100).toString())
          fetchBids()
        }
      }

      ws.onerror = () => {
        addLog('ERROR', 'WebSocket error')
      }
    }

    connect()

    return () => {
      isUnmounted = true
      if (wsRef.current) {
        wsRef.current.close()
      }
      clearTimeout(reconnectTimeout)
    }
  }, [tenderId, addLog])

  useEffect(() => {
    if (logContainerRef.current) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight
    }
  }, [logs])

  useEffect(() => {
    let interval: ReturnType<typeof setInterval>
    if (cooldown > 0) {
      interval = setInterval(() => {
        const remaining = Math.max(0, 1000 - (Date.now() - lastBidTime))
        setCooldown(remaining)
        if (remaining === 0) {
          clearInterval(interval)
        }
      }, 50)
    }
    return () => clearInterval(interval)
  }, [cooldown, lastBidTime])

  const handlePlaceBid = (amount?: number) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return
    if (cooldown > 0) return
    
    let bidInCents: number
    if (amount !== undefined) {
      bidInCents = amount
    } else {
      const bidValue = parseFloat(bidAmount)
      if (isNaN(bidValue)) return
      bidInCents = Math.round(bidValue * 100)
    }

    const payload = {
      type: "place_bid",
      bid: bidInCents,
      companyId: companyId,
      personId: personId
    }

    addLog('SEND', payload)
    wsRef.current.send(JSON.stringify(payload))
    setLastBidTime(Date.now())
    setCooldown(1000)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <Button variant="secondary" size="sm" onClick={() => navigate('/ui/auctions')}>
            <ArrowLeft className="mr-2 h-4 w-4" size={16} />
            Back
          </Button>
          <h1 className="text-3xl font-bold">
            Auction: <Code className="text-2xl px-2">{tenderId}</Code>
          </h1>
        </div>
        <Button variant="outline" size="sm" onClick={fetchBids} disabled={isRefreshingBids}>
          <RefreshCw className={cn("mr-2 h-4 w-4", isRefreshingBids && "animate-spin")} size={16} />
          Refresh
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Users className="h-5 w-5" />
                Participants
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {participants.length === 0 ? (
                  <div className="text-center py-4 text-muted-foreground">
                    No participants registered.
                  </div>
                ) : (
                  participants.map((p, idx) => (
                    <div key={idx} className="flex items-center gap-2 p-2 border rounded bg-muted/30 font-mono text-sm">
                      <Building2 className="h-4 w-4 text-muted-foreground" />
                      {p}
                    </div>
                  ))
                )}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Bid History</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="relative overflow-x-auto rounded-lg border">
                <table className="w-full text-sm text-left">
                  <thead className="text-xs uppercase bg-muted">
                    <tr>
                      <th className="px-4 py-3">Time</th>
                      <th className="px-4 py-3">Bidder (Company/Person)</th>
                      <th className="px-4 py-3 text-right">Amount</th>
                    </tr>
                  </thead>
                  <tbody>
                    {bids.map((bid) => (
                      <tr key={bid.id} className="border-t hover:bg-muted/50 transition-colors">
                        <td className="px-4 py-3 text-muted-foreground whitespace-nowrap">
                          {new Date(bid.createdAt).toLocaleTimeString()}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex flex-col gap-1">
                            <div className="flex items-center gap-2 text-primary font-semibold">
                              <Building2 className="h-3.5 w-3.5 text-muted-foreground" />
                              <span>{bid.companyId}</span>
                            </div>
                            <div className="flex items-center gap-2 text-xs text-muted-foreground">
                              <User className="h-3 w-3" />
                              <span className="italic">{bid.personId}</span>
                            </div>
                          </div>
                        </td>
                        <td className="px-4 py-3 text-right font-bold text-primary">
                          {formatCurrency(bid.bidAmount)}
                        </td>
                      </tr>
                    ))}
                    {bids.length === 0 && (
                      <tr>
                        <td colSpan={3} className="px-4 py-8 text-center text-muted-foreground">
                          No bids placed yet.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Live Events (Debug)</CardTitle>
            </CardHeader>
            <CardContent>
              <pre 
                ref={logContainerRef}
                className="bg-muted p-4 rounded-lg font-mono text-xs h-[300px] overflow-y-auto whitespace-pre-wrap"
              >
                {logs.map((log, i) => (
                  <div key={i} className="mb-1">
                    <span className="text-muted-foreground">[{log.time}]</span>{" "}
                    <span className="font-bold text-primary">{log.type}:</span>{" "}
                    {JSON.stringify(log.data)}
                  </div>
                ))}
                {logs.length === 0 && <span className="text-muted-foreground">No events yet...</span>}
              </pre>
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card>
            <CardContent className="pt-6">
              <div className="flex justify-between items-center mb-4">
                <Badge variant={wsStatus === 'connected' ? 'success' : wsStatus === 'connecting' ? 'warning' : 'destructive'}>
                  {wsStatus}
                </Badge>
              </div>
              
              <div className="space-y-1 mb-6">
                <div className="text-xs uppercase tracking-wider text-muted-foreground">
                  Last Bid / Current Price
                </div>
                <div className="text-4xl font-bold text-primary">
                  {formatCurrency(currentPrice)}
                </div>
                <div className="text-sm text-muted-foreground">
                  {winnerId ? (
                    <>Winning: <Code>{winnerId}</Code></>
                  ) : (
                    'No bids yet'
                  )}
                </div>
              </div>

              <div className="space-y-4 pt-4 border-t">
                <h3 className="font-semibold">Place a Bid</h3>
                <div className="space-y-2">
                  <label htmlFor="bid-amount" className="text-sm text-muted-foreground">Your Bid</label>
                  <div className="relative">
                    <span className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground text-sm">$</span>
                    <Input 
                      id="bid-amount" 
                      type="number" 
                      step="0.01"
                      placeholder="Enter amount"
                      className="pl-7"
                      value={bidAmount}
                      onChange={(e) => setBidAmount(e.target.value)}
                    />
                  </div>
                </div>

                {!isFinished && currentPrice !== null && (
                  <div className="flex gap-2">
                    {[1, 2, 3].map(multiplier => {
                      const amount = currentPrice - (step * multiplier)
                      if (amount <= 0) return null
                      return (
                        <Button
                          key={multiplier}
                          variant="outline"
                          size="sm"
                          className="flex-1 border-red-500 text-red-500 hover:bg-red-50 hover:text-red-600 transition-colors"
                          onClick={() => handlePlaceBid(amount)}
                          disabled={wsStatus !== 'connected' || cooldown > 0}
                        >
                          <span>$</span>
                          <span className="pl-4">{(amount / 100).toFixed(2)}</span>
                        </Button>
                      )
                    })}
                  </div>
                )}

                <Button 
                  className="w-full relative overflow-hidden" 
                  onClick={() => handlePlaceBid()}
                  disabled={wsStatus !== 'connected' || cooldown > 0 || isFinished}
                >
                  {cooldown > 0 && (
                    <div 
                      className="absolute left-0 top-0 h-full bg-primary-foreground/20 transition-all duration-50 ease-linear"
                      style={{ width: `${(cooldown / 1000) * 100}%` }}
                    />
                  )}
                  <Send className="mr-2 h-4 w-4" size={16} />
                  {cooldown > 0 ? `Wait ${Math.ceil(cooldown / 100) / 10}s` : 'Place Bid'}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}

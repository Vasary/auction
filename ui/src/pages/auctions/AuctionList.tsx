import { useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "../../components/Card"
import { Button } from "../../components/Button"
import { Badge } from "../../components/Badge"
import { Code } from "../../components/Code"
import { RefreshCw, LogOut } from "lucide-react"
import { format } from "date-fns"
import { cn, formatCurrency } from "../../lib/utils"

interface Auction {
  tenderId: string
  status: 'Active' | 'Scheduled' | 'Finished'
  currentPrice: number
  startAt: string
  endAt: string
}

function CountdownTimer({ targetDate }: { targetDate: string }) {
  const [timeLeft, setTimeLeft] = useState<string>("")

  useEffect(() => {
    const calculateTimeLeft = () => {
      const difference = new Date(targetDate).getTime() - new Date().getTime()
      
      if (difference <= 0) {
        setTimeLeft("00:00:00")
        return
      }

      const hours = Math.floor(difference / (1000 * 60 * 60))
      const minutes = Math.floor((difference / 1000 / 60) % 60)
      const seconds = Math.floor((difference / 1000) % 60)

      setTimeLeft(
        `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
      )
    }

    calculateTimeLeft()
    const timer = setInterval(calculateTimeLeft, 1000)

    return () => clearInterval(timer)
  }, [targetDate])

  return (
    <span className="font-mono font-bold text-red-600">
      {timeLeft}
    </span>
  )
}

interface AuctionListProps {
  personId: string
  companyId: string
  onLogout: () => void
}

export default function AuctionList({ personId, companyId, onLogout }: AuctionListProps) {
  const [auctions, setAuctions] = useState<Auction[]>([])
  const [loading, setLoading] = useState(true)

  const loadAuctions = async () => {
    setLoading(true)
    try {
      const res = await fetch('/auctions')
      const data = await res.json()
      setAuctions(data)
    } catch (err) {
      console.error('Failed to load auctions', err)
    } finally {
      setLoading(false)
    }
  }

  const handleParticipate = async (tenderId: string) => {
    try {
      const res = await fetch(`/auctions/${encodeURIComponent(tenderId)}/participate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ companyId })
      })

      if (res.ok) {
        alert('Successfully registered as a participant!')
      } else {
        const errData = await res.json().catch(() => ({}))
        alert(`Failed to participate: ${errData.error || res.statusText}`)
      }
    } catch (err) {
      console.error('Participation error', err)
      alert('An error occurred. Check console for details.')
    }
  }

  useEffect(() => {
    loadAuctions()
  }, [])

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold">Auctions</h1>
          <p className="text-muted-foreground mt-1">
            Logged in as <Code>{personId}</Code> (Company: <Code>{companyId}</Code>)
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={loadAuctions} disabled={loading}>
            <RefreshCw className={cn("mr-2 h-4 w-4", loading && "animate-spin")} size={16} />
            Refresh
          </Button>
          <Button variant="secondary" size="sm" onClick={onLogout}>
            <LogOut className="mr-2 h-4 w-4" size={16} />
            Logout
          </Button>
        </div>
      </div>

      {loading && auctions.length === 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3].map(i => (
            <Card key={i} className="animate-pulse">
              <div className="h-48 bg-muted rounded-xl"></div>
            </Card>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {auctions.map((auction) => (
            <Card key={auction.tenderId} className="flex flex-col">
              <CardHeader className="pb-2">
                <div className="flex justify-between items-center mb-2">
                  <Badge variant={
                    auction.status === 'Active' ? 'success' : 
                    auction.status === 'Scheduled' ? 'warning' : 'default'
                  }>
                    {auction.status}
                  </Badge>
                  {auction.status === 'Active' && (
                    <div className="flex items-center gap-1 animate-pulse bg-red-50 px-1.5 py-0.5 rounded-full border border-red-100 text-xs shadow-sm">
                      <CountdownTimer targetDate={auction.endAt} />
                    </div>
                  )}
                </div>
                <CardTitle className="font-mono text-xl">{auction.tenderId}</CardTitle>
              </CardHeader>
              <CardContent className="flex-1 pb-2">
                <div className="text-xs uppercase tracking-wider text-muted-foreground mb-1">
                  Last Bid / Current Price
                </div>
                <div className="text-3xl font-bold text-primary">
                  {formatCurrency(auction.currentPrice)}
                </div>
                <div className="text-sm text-muted-foreground mt-4 space-y-2">
                  <div className="flex justify-between items-center">
                    <span>Starts: {format(new Date(auction.startAt), "PPpp")}</span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span>Ends: {format(new Date(auction.endAt), "PPpp")}</span>
                  </div>
                </div>
              </CardContent>
              <CardFooter className="pt-4 gap-2">
                <Link to={`/ui/auctions/${auction.tenderId}`} className="flex-1">
                  <Button className="w-full">Join</Button>
                </Link>
                {auction.status !== 'Finished' && (
                  <Button 
                    variant="outline" 
                    className="flex-1 border-emerald-200 text-emerald-700 hover:bg-emerald-50 hover:text-emerald-800"
                    onClick={() => handleParticipate(auction.tenderId)}
                  >
                    Participate
                  </Button>
                )}
              </CardFooter>
            </Card>
          ))}
          {auctions.length === 0 && !loading && (
            <div className="col-span-full py-12 text-center border rounded-xl border-dashed">
               <p className="text-muted-foreground">No auctions found.</p>
               <Button variant="link" onClick={loadAuctions}>Try again</Button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

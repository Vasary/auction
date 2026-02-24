import { Link } from "react-router-dom"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/Card"
import { Activity, LayoutDashboard, PenTool } from "lucide-react"

export default function LandingPage() {
  return (
    <div className="container mx-auto px-4 py-12 flex flex-col items-center justify-center min-h-[80vh]">
      <h1 className="text-4xl font-bold mb-4 text-center">Agora Auction System</h1>
      <p className="text-muted-foreground text-xl mb-12 text-center max-w-2xl">
        Choose your interface to manage auctions or generate requests for testing.
      </p>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-8 w-full max-w-6xl">
        <Link to="/ui/auctions" className="group">
          <Card className="h-full transition-all group-hover:shadow-lg group-hover:border-primary">
            <CardHeader>
              <div className="w-12 h-12 bg-primary/10 text-primary rounded-lg flex items-center justify-center mb-4 group-hover:bg-primary group-hover:text-white transition-colors">
                <LayoutDashboard size={24} />
              </div>
              <CardTitle className="text-2xl">Developer UI</CardTitle>
              <CardDescription>
                Dashboard for managing and participating in live auctions.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="text-sm text-muted-foreground space-y-2">
                <li>• View active auctions</li>
                <li>• Real-time price updates</li>
                <li>• Place bids via WebSocket</li>
                <li>• Participant registration</li>
              </ul>
            </CardContent>
          </Card>
        </Link>

        <Link to="/ui/generator" className="group">
          <Card className="h-full transition-all group-hover:shadow-lg group-hover:border-primary">
            <CardHeader>
              <div className="w-12 h-12 bg-primary/10 text-primary rounded-lg flex items-center justify-center mb-4 group-hover:bg-primary group-hover:text-white transition-colors">
                <PenTool size={24} />
              </div>
              <CardTitle className="text-2xl">Request Generator</CardTitle>
              <CardDescription>
                Tool for manual testing of HTTP endpoints and WebSocket events.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="text-sm text-muted-foreground space-y-2">
                <li>• CRUD operations for auctions</li>
                <li>• UUIDv7 generation tool</li>
                <li>• WebSocket event playground</li>
                <li>• Health check monitoring</li>
              </ul>
            </CardContent>
          </Card>
        </Link>

        <Link to="/ui/e2e" className="group">
          <Card className="h-full transition-all group-hover:shadow-lg group-hover:border-primary">
            <CardHeader>
              <div className="w-12 h-12 bg-primary/10 text-primary rounded-lg flex items-center justify-center mb-4 group-hover:bg-primary group-hover:text-white transition-colors">
                <Activity size={24} />
              </div>
              <CardTitle className="text-2xl">E2E Runner</CardTitle>
              <CardDescription>
                Automated functional scenario runner with multiple auctions and clients.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="text-sm text-muted-foreground space-y-2">
                <li>• Creates multiple auctions</li>
                <li>• Connects 3-4 WS clients per auction</li>
                <li>• Simulates bids and waits for finish</li>
                <li>• Validates winners and final prices</li>
              </ul>
            </CardContent>
          </Card>
        </Link>
      </div>
    </div>
  )
}

import { Activity, Home } from "lucide-react"
import { Link } from "react-router-dom"
import { Button } from "../components/Button"
import E2ERunnerPanel from "../components/E2ERunnerPanel"

export default function E2ERunnerPage() {
  return (
    <div className="container mx-auto px-4 py-8 space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Activity className="h-6 w-6 text-emerald-500" />
            E2E Runner
          </h1>
          <p className="text-muted-foreground">Automated multi-auction functional testing</p>
        </div>
        <div className="flex gap-2">
          <Link to="/ui">
            <Button variant="outline">
              <Home className="mr-2 h-4 w-4" />
              Home
            </Button>
          </Link>
        </div>
      </div>

      <E2ERunnerPanel />
    </div>
  )
}

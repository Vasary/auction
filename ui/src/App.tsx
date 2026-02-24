import { BrowserRouter, Routes, Route } from "react-router-dom"
import LandingPage from "./pages/LandingPage"
import RequestGenerator from "./pages/RequestGenerator"
import AuctionsPage from "./pages/auctions/AuctionsPage"
import E2ERunnerPage from "./pages/E2ERunnerPage"

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-background">
        <Routes>
          <Route path="/ui" element={<LandingPage />} />
          <Route path="/ui/auctions/*" element={<AuctionsPage />} />
          <Route path="/ui/generator" element={<RequestGenerator />} />
          <Route path="/ui/e2e" element={<E2ERunnerPage />} />
        </Routes>
      </div>
    </BrowserRouter>
  )
}

export default App

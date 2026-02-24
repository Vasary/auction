import { useState } from "react"
import { Routes, Route, Navigate } from "react-router-dom"
import AuctionLogin from "./AuctionLogin"
import AuctionList from "./AuctionList"
import AuctionDetail from "./AuctionDetail"

export default function AuctionsPage() {
  const [personId, setPersonId] = useState<string | null>(localStorage.getItem('personId'))
  const [companyId, setCompanyId] = useState<string | null>(localStorage.getItem('companyId'))

  const handleLogin = (pId: string, cId: string) => {
    setPersonId(pId)
    setCompanyId(cId)
    localStorage.setItem('personId', pId)
    localStorage.setItem('companyId', cId)
  }

  const handleLogout = () => {
    setPersonId(null)
    setCompanyId(null)
    localStorage.removeItem('personId')
    localStorage.removeItem('companyId')
  }

  return (
    <div className="container mx-auto px-4 py-8 min-h-screen">
      <Routes>
        <Route 
          path="/" 
          element={
            personId && companyId ? (
              <AuctionList personId={personId} companyId={companyId} onLogout={handleLogout} />
            ) : (
              <AuctionLogin onLogin={handleLogin} />
            )
          } 
        />
        <Route 
          path="/:tenderId" 
          element={
            personId && companyId ? (
              <AuctionDetail personId={personId} companyId={companyId} />
            ) : (
              <Navigate to="/ui/auctions" replace />
            )
          } 
        />
      </Routes>
    </div>
  )
}

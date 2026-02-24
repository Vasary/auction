import { useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/Card"
import { Input } from "../../components/Input"
import { Button } from "../../components/Button"

interface LoginProps {
  onLogin: (personId: string, companyId: string) => void
}

export default function AuctionLogin({ onLogin }: LoginProps) {
  const [personId, setPersonId] = useState("")
  const [companyId, setCompanyId] = useState("")

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (personId && companyId) {
      onLogin(personId, companyId)
    }
  }

  return (
    <div className="flex items-center justify-center min-h-[60vh]">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-2xl">Welcome to Auctions</CardTitle>
          <CardDescription>Please enter your details to continue</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <label htmlFor="person-id" className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                Person ID
              </label>
              <Input
                id="person-id"
                placeholder="e.g. person-123"
                value={personId}
                onChange={(e) => setPersonId(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <label htmlFor="company-id" className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                Company ID
              </label>
              <Input
                id="company-id"
                placeholder="e.g. company-456"
                value={companyId}
                onChange={(e) => setCompanyId(e.target.value)}
                required
              />
            </div>
            <Button type="submit" className="w-full">
              Enter
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

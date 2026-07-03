import { useEffect, useState } from "react"

export function useAuth() {
  const [token, setToken] = useState<string | null>(localStorage.getItem("token"))

  useEffect(() => {
    const handler = () => setToken(localStorage.getItem("token"))
    window.addEventListener("storage", handler)
    return () => window.removeEventListener("storage", handler)
  }, [])

  const login = (newToken: string) => {
    localStorage.setItem("token", newToken)
    setToken(newToken)
  }

  const logout = () => {
    localStorage.removeItem("token")
    setToken(null)
  }

  return { token, isAuthed: !!token, login, logout }
}

import { useEffect, useState } from "react"
import { RefreshCw, Trash2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  clearFirewallBlacklist,
  deleteFirewallBlacklistEntry,
  type FirewallBlacklistEntry,
  listFirewallBlacklist,
} from "@/api/admin"

export function FirewallBlacklistPage() {
  const [blacklist, setBlacklist] = useState<FirewallBlacklistEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [clearOpen, setClearOpen] = useState(false)
  const [clearing, setClearing] = useState(false)
  const [deletingKey, setDeletingKey] = useState("")

  const fetchBlacklist = async () => {
    setLoading(true)
    try {
      const res = await listFirewallBlacklist()
      setBlacklist(res.data.data || [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchBlacklist()
  }, [])

  const handleDeleteEntry = async (entry: FirewallBlacklistEntry) => {
    if (!confirm(`确认删除规则 ${entry.rule_id} 的这条风控记录？`)) return
    setDeletingKey(entry.key)
    try {
      await deleteFirewallBlacklistEntry(entry.key)
      fetchBlacklist()
    } catch (err: any) {
      alert(err.response?.data?.message || "删除失败")
    } finally {
      setDeletingKey("")
    }
  }

  const handleClearBlacklist = async () => {
    setClearing(true)
    try {
      await clearFirewallBlacklist()
      setClearOpen(false)
      fetchBlacklist()
    } catch (err: any) {
      alert(err.response?.data?.message || "清空失败")
    } finally {
      setClearing(false)
    }
  }

  const formatTTL = (seconds: number) => {
    if (seconds === -1) return "永久"
    if (seconds === -2) return "已过期"
    if (seconds < 60) return `${seconds} 秒`
    if (seconds < 3600) return `${Math.ceil(seconds / 60)} 分钟`
    return `${Math.ceil(seconds / 3600)} 小时`
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">风控名单</h1>
          <p className="text-sm text-muted-foreground">查看和清理 PostgreSQL 持久化保存的风控计数</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" onClick={fetchBlacklist} disabled={loading}>
            <RefreshCw className={loading ? "animate-spin" : ""} />
            {loading ? "刷新中..." : "刷新"}
          </Button>
          <Button variant="destructive" onClick={() => setClearOpen(true)} disabled={blacklist.length === 0 || clearing}>
            <Trash2 />
            全部清空
          </Button>
        </div>
      </div>

      <Card className="border-border/60 shadow-sm">
        <CardHeader>
          <CardTitle>当前风控名单</CardTitle>
          <CardDescription>Redis 仅作为运行时缓存，删除或清空会同步处理持久化记录和缓存</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="py-8 text-center text-sm text-muted-foreground">加载中...</div>
          ) : blacklist.length === 0 ? (
            <div className="py-8 text-center text-sm text-muted-foreground">暂无风控名单</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>规则 ID</TableHead>
                  <TableHead>身份标识</TableHead>
                  <TableHead className="text-right">计数</TableHead>
                  <TableHead>过期时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {blacklist.map((entry) => (
                  <TableRow key={entry.key}>
                    <TableCell>{entry.rule_id || "-"}</TableCell>
                    <TableCell>
                      <div className="max-w-[720px] break-all font-mono text-xs">{entry.identity || entry.key}</div>
                    </TableCell>
                    <TableCell className="text-right">{entry.count}</TableCell>
                    <TableCell>{formatTTL(entry.ttl_seconds)}</TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDeleteEntry(entry)}
                        disabled={deletingKey === entry.key}
                        aria-label="删除风控记录"
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={clearOpen} onOpenChange={setClearOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>清空当前风控名单</DialogTitle>
            <DialogDescription>
              这会删除持久化风控名单及对应 Redis 缓存，已被拦截的身份会重新获得一次提交机会。管理员登录状态不会受影响。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setClearOpen(false)} disabled={clearing}>
              取消
            </Button>
            <Button variant="destructive" onClick={handleClearBlacklist} disabled={clearing}>
              {clearing ? "清空中..." : "确认清空"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

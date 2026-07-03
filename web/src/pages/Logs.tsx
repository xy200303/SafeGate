import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { listLogs, type ProxyLog } from "@/api/admin"

export function LogsPage() {
  const [logs, setLogs] = useState<ProxyLog[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const pageSize = 20

  const fetch = async () => {
    try {
      const res = await listLogs(page, pageSize)
      setLogs(res.data.data?.list || [])
      setTotal(res.data.data?.total || 0)
    } catch (err) {
      console.error(err)
    }
  }

  useEffect(() => {
    fetch()
  }, [page])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">访问日志</h1>
        <p className="text-sm text-muted-foreground">查看代理请求与风控拦截记录</p>
      </div>

      <Card className="border-border/60 shadow-sm">
        <CardHeader>
          <CardTitle>日志列表</CardTitle>
          <CardDescription>最近经过 SafeGate 的代理请求</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>时间</TableHead>
                <TableHead>域名</TableHead>
                <TableHead>IP</TableHead>
                <TableHead>方法</TableHead>
                <TableHead>路径</TableHead>
                <TableHead>目标</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>拦截</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="whitespace-nowrap text-muted-foreground">{new Date(log.created_at).toLocaleString()}</TableCell>
                  <TableCell className="font-medium">{log.bind_domain}</TableCell>
                  <TableCell>{log.client_ip}</TableCell>
                  <TableCell>{log.method}</TableCell>
                  <TableCell className="max-w-xs truncate">{log.path}</TableCell>
                  <TableCell className="max-w-xs truncate text-muted-foreground">{log.target_url}</TableCell>
                  <TableCell>{log.status_code ?? "-"}</TableCell>
                  <TableCell>
                    {log.blocked ? (
                      <Badge variant="destructive">是</Badge>
                    ) : (
                      <Badge variant="outline">否</Badge>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {logs.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8} className="py-8 text-center text-muted-foreground">
                    暂无日志
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>

          <div className="mt-4 flex items-center justify-between">
            <Button variant="outline" disabled={page <= 1} onClick={() => setPage(page - 1)}>
              上一页
            </Button>
            <span className="text-sm text-muted-foreground">
              第 {page} / {totalPages} 页，共 {total} 条
            </span>
            <Button variant="outline" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>
              下一页
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

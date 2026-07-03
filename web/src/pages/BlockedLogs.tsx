import { useEffect, useState } from "react"
import { Eye } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { listBlockedLogs, type ProxyLog } from "@/api/admin"

function formatJSON(v: unknown): string {
  try {
    return JSON.stringify(v, null, 2)
  } catch {
    return String(v)
  }
}

function BlockDetail({ log }: { log: ProxyLog }) {
  return (
    <div className="space-y-4 text-sm">
      <div className="grid grid-cols-2 gap-3">
        <div className="rounded-md bg-muted p-3">
          <div className="text-muted-foreground">客户端 IP</div>
          <div className="font-medium">{log.client_ip}</div>
        </div>
        <div className="rounded-md bg-muted p-3">
          <div className="text-muted-foreground">请求方法</div>
          <div className="font-medium">{log.method}</div>
        </div>
        <div className="rounded-md bg-muted p-3">
          <div className="text-muted-foreground">状态码</div>
          <div className="font-medium">{log.status_code ?? "-"}</div>
        </div>
        <div className="rounded-md bg-muted p-3">
          <div className="text-muted-foreground">触发时间</div>
          <div className="font-medium">{new Date(log.created_at).toLocaleString()}</div>
        </div>
      </div>

      <div className="rounded-md bg-muted p-3">
        <div className="text-muted-foreground">请求路径</div>
        <div className="mt-1 break-all font-mono">{log.path}</div>
      </div>

      {(log.rule_name || log.rule_id) && (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-900/50 dark:bg-red-950/20">
          <div className="text-red-700 dark:text-red-400">触发规则</div>
          <div className="mt-1 font-medium">
            {log.rule_name || "未知规则"}
            {log.rule_id ? ` (#${log.rule_id})` : ""}
          </div>
        </div>
      )}

      {log.message && (
        <div className="rounded-md bg-muted p-3">
          <div className="text-muted-foreground">拦截原因</div>
          <div className="mt-1 font-medium text-red-600 dark:text-red-400">{log.message}</div>
        </div>
      )}

      {log.user_agent && (
        <div className="rounded-md bg-muted p-3">
          <div className="text-muted-foreground">User-Agent</div>
          <div className="mt-1 break-all font-mono text-xs">{log.user_agent}</div>
        </div>
      )}

      {log.query_params && Object.keys(log.query_params).length > 0 && (
        <div>
          <div className="mb-1 text-muted-foreground">查询参数</div>
          <pre className="max-h-40 overflow-auto rounded-md bg-muted p-3 text-xs">
            {formatJSON(log.query_params)}
          </pre>
        </div>
      )}

      {log.request_headers && Object.keys(log.request_headers).length > 0 && (
        <div>
          <div className="mb-1 text-muted-foreground">请求头</div>
          <pre className="max-h-48 overflow-auto rounded-md bg-muted p-3 text-xs">
            {formatJSON(log.request_headers)}
          </pre>
        </div>
      )}

      {log.request_body && (
        <div>
          <div className="mb-1 text-muted-foreground">请求体</div>
          <pre className="max-h-60 overflow-auto rounded-md bg-muted p-3 text-xs">
            {log.request_body}
          </pre>
        </div>
      )}
    </div>
  )
}

export function BlockedLogsPage() {
  const [logs, setLogs] = useState<ProxyLog[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const pageSize = 20

  const fetch = async () => {
    try {
      const res = await listBlockedLogs(page, pageSize)
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
        <h1 className="text-2xl font-bold tracking-tight">拦截日志</h1>
        <p className="text-sm text-muted-foreground">只记录被 SafeGate 拦截的请求，包含触发规则与详细请求数据</p>
      </div>

      <Card className="border-border/60 shadow-sm">
        <CardHeader>
          <CardTitle>拦截记录</CardTitle>
          <CardDescription>按时间倒序展示最近被拦截的请求</CardDescription>
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
                <TableHead>状态</TableHead>
                <TableHead>触发规则</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="whitespace-nowrap text-muted-foreground">
                    {new Date(log.created_at).toLocaleString()}
                  </TableCell>
                  <TableCell className="font-medium">{log.bind_domain}</TableCell>
                  <TableCell>{log.client_ip}</TableCell>
                  <TableCell>{log.method}</TableCell>
                  <TableCell className="max-w-xs truncate">{log.path}</TableCell>
                  <TableCell>{log.status_code ?? "-"}</TableCell>
                  <TableCell>
                    <Badge variant="destructive">{log.rule_name || "未知规则"}</Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Dialog>
                      <DialogTrigger asChild>
                        <Button variant="ghost" size="sm" className="h-8 gap-1">
                          <Eye className="h-4 w-4" />
                          详情
                        </Button>
                      </DialogTrigger>
                      <DialogContent className="max-h-[90vh] max-w-2xl overflow-auto">
                        <DialogHeader>
                          <DialogTitle>拦截详情</DialogTitle>
                          <DialogDescription>
                            {log.bind_domain} · {log.client_ip}
                          </DialogDescription>
                        </DialogHeader>
                        <BlockDetail log={log} />
                      </DialogContent>
                    </Dialog>
                  </TableCell>
                </TableRow>
              ))}
              {logs.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8} className="py-8 text-center text-muted-foreground">
                    暂无拦截日志
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

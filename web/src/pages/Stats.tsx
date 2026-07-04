import { useEffect, useState } from "react"
import { BarChart3, RefreshCw, Shield, UserX, Users } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { type BlockedStats, getBlockedStats } from "@/api/admin"

export function StatsPage() {
  const [stats, setStats] = useState<BlockedStats | null>(null)
  const [loading, setLoading] = useState(false)

  const fetch = async () => {
    setLoading(true)
    try {
      const res = await getBlockedStats()
      if (res.data.data) {
        setStats(res.data.data)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetch()
  }, [])

  const maxIPCount = stats?.top_ips?.[0]?.count || 1
  const maxRuleCount = stats?.top_rules?.[0]?.count || 1
  const maxTrendCount = Math.max(1, ...(stats?.daily_trend?.map((d) => d.count) || [1]))

  const StatCard = ({
    label,
    value,
    icon: Icon,
    color,
  }: {
    label: string
    value: number
    icon: React.ElementType
    color: string
  }) => (
    <Card className="border-border/60 shadow-sm">
      <CardContent className="flex items-center justify-between p-6">
        <div>
          <p className="text-sm font-medium text-muted-foreground">{label}</p>
          <p className="mt-2 text-3xl font-bold tracking-tight">{value.toLocaleString()}</p>
        </div>
        <div className={`flex h-12 w-12 items-center justify-center rounded-full ${color}`}>
          <Icon className="h-6 w-6 text-white" />
        </div>
      </CardContent>
    </Card>
  )

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">首页</h1>
          <p className="text-sm text-muted-foreground">按被封锁客户端 IP 和规则维度聚合的数据概览</p>
        </div>
        <Button variant="outline" onClick={fetch} disabled={loading}>
          <RefreshCw className={loading ? "animate-spin" : ""} />
          {loading ? "刷新中..." : "刷新"}
        </Button>
      </div>

      {stats && (
        <>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <StatCard
              label="总拦截次数"
              value={stats.total_blocked}
              icon={Shield}
              color="bg-red-500"
            />
            <StatCard
              label="今日拦截"
              value={stats.today_blocked}
              icon={BarChart3}
              color="bg-orange-500"
            />
            <StatCard
              label="被拦截 IP 数"
              value={stats.unique_ips}
              icon={Users}
              color="bg-rose-500"
            />
            <StatCard
              label="活跃拦截规则"
              value={stats.active_rules}
              icon={UserX}
              color="bg-pink-500"
            />
          </div>

          <div className="grid gap-6 lg:grid-cols-2">
            <Card className="border-border/60 shadow-sm">
              <CardHeader>
                <CardTitle>TOP 10 被拦截 IP</CardTitle>
                <CardDescription>按拦截次数排序</CardDescription>
              </CardHeader>
              <CardContent>
                {stats.top_ips.length === 0 ? (
                  <div className="py-8 text-center text-sm text-muted-foreground">暂无数据</div>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>客户端 IP</TableHead>
                        <TableHead className="text-right">拦截次数</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {stats.top_ips.map((item) => (
                        <TableRow key={item.client_ip}>
                          <TableCell>
                            <div className="font-medium">{item.client_ip}</div>
                            <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-muted">
                              <div
                                className="h-full rounded-full bg-red-500"
                                style={{ width: `${Math.min(100, (item.count / maxIPCount) * 100)}%` }}
                              />
                            </div>
                          </TableCell>
                          <TableCell className="text-right align-top">{item.count}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>

            <Card className="border-border/60 shadow-sm">
              <CardHeader>
                <CardTitle>TOP 触发规则</CardTitle>
                <CardDescription>按规则触发次数排序</CardDescription>
              </CardHeader>
              <CardContent>
                {stats.top_rules.length === 0 ? (
                  <div className="py-8 text-center text-sm text-muted-foreground">暂无数据</div>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>规则名称</TableHead>
                        <TableHead className="text-right">触发次数</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {stats.top_rules.map((item) => (
                        <TableRow key={item.rule_id}>
                          <TableCell>
                            <div className="font-medium">{item.rule_name}</div>
                            <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-muted">
                              <div
                                className="h-full rounded-full bg-orange-500"
                                style={{ width: `${Math.min(100, (item.count / maxRuleCount) * 100)}%` }}
                              />
                            </div>
                          </TableCell>
                          <TableCell className="text-right align-top">{item.count}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>
          </div>

          <Card className="border-border/60 shadow-sm">
            <CardHeader>
              <CardTitle>近 7 天拦截趋势</CardTitle>
              <CardDescription>每日被拦截次数变化</CardDescription>
            </CardHeader>
            <CardContent>
              {stats.daily_trend.length === 0 ? (
                <div className="py-8 text-center text-sm text-muted-foreground">暂无数据</div>
              ) : (
                <div className="flex items-end gap-2 md:gap-4">
                  {stats.daily_trend.map((item) => (
                    <div key={item.date} className="flex flex-1 flex-col items-center gap-2">
                      <div className="relative w-full">
                        <div
                          className="w-full rounded-t-md bg-gradient-to-t from-red-500 to-red-400"
                          style={{ height: `${Math.max(24, (item.count / maxTrendCount) * 160)}px` }}
                        />
                        <div className="absolute -top-6 left-1/2 -translate-x-1/2 text-xs font-medium text-muted-foreground">
                          {item.count}
                        </div>
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {item.date.slice(5)}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </>
      )}

      {!stats && !loading && (
        <Card className="border-border/60 shadow-sm">
          <CardContent className="py-12 text-center text-muted-foreground">
            加载失败，请点击刷新重试
          </CardContent>
        </Card>
      )}
    </div>
  )
}

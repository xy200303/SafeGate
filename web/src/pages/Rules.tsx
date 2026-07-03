import { useEffect, useState } from "react"
import { useSearchParams } from "react-router-dom"
import { Edit, Plus, Trash2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { type Rule, listRules, createRule, updateRule, deleteRule } from "@/api/admin"

const emptyRule: Partial<Rule> = {
  name: "",
  path_prefix: "/api/register",
  methods: "POST",
  rule_type: "duplicate_ip",
  identity_fields: "",
  max_attempts: 1,
  window_seconds: 0,
  block_seconds: 0,
  block_status: 403,
  block_response: '{"code":403,"message":"重复注册"}',
  enabled: true,
}

const ruleTemplates = [
  {
    label: "单 IP 仅一次成功",
    form: { ...emptyRule, name: "单IP单次注册", identity_fields: "" },
  },
  {
    label: "IP + 手机号唯一",
    form: { ...emptyRule, name: "IP手机号防重", identity_fields: "phone" },
  },
  {
    label: "IP + 手机号 + 姓名 + 收款账户",
    form: { ...emptyRule, name: "多字段组合防重", identity_fields: "phone,name,bank_account" },
  },
  {
    label: "IP 注册速率限制",
    form: {
      ...emptyRule,
      name: "注册速率限制",
      rule_type: "rate_limit" as Rule["rule_type"],
      identity_fields: "",
      max_attempts: 10,
      window_seconds: 60,
      block_response: '{"code":429,"message":"请求过于频繁"}',
    },
  },
]

export function RulesPage() {
  const [searchParams] = useSearchParams()
  const domainId = Number(searchParams.get("domain_id") || 0)
  const [rules, setRules] = useState<Rule[]>([])
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Rule | null>(null)
  const [form, setForm] = useState<Partial<Rule>>(emptyRule)

  const fetch = async () => {
    if (!domainId) return
    setLoading(true)
    try {
      const res = await listRules(domainId)
      setRules(res.data.data || [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetch()
  }, [domainId])

  const openCreate = () => {
    setEditing(null)
    setForm({ ...emptyRule, domain_id: domainId })
    setOpen(true)
  }

  const openEdit = (r: Rule) => {
    setEditing(r)
    setForm({ ...r })
    setOpen(true)
  }

  const normalizeJSON = (value: string) => {
    try {
      return value ? JSON.parse(value) : {}
    } catch {
      return value
    }
  }

  const handleSave = async () => {
    try {
      const payload = {
        ...form,
        block_response: normalizeJSON(form.block_response || ""),
      }
      if (editing) {
        await updateRule(editing.id, payload)
      } else {
        await createRule(payload)
      }
      setOpen(false)
      fetch()
    } catch (err: any) {
      alert(err.response?.data?.message || "保存失败")
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm("确认删除？")) return
    try {
      await deleteRule(id)
      fetch()
    } catch (err: any) {
      alert(err.response?.data?.message || "删除失败")
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">接口风控规则</h1>
          <p className="text-sm text-muted-foreground">域名 ID: {domainId || "未选择"}</p>
        </div>
        <Button onClick={openCreate} disabled={!domainId}>
          <Plus className="mr-2 h-4 w-4" />
          添加规则
        </Button>
      </div>

      <Card className="border-border/60 shadow-sm">
        <CardHeader>
          <CardTitle>规则列表</CardTitle>
          <CardDescription>针对该域名的重复 IP / 速率限制规则</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="py-8 text-center text-muted-foreground">加载中...</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>名称</TableHead>
                  <TableHead>路径</TableHead>
                  <TableHead>方法</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>次数</TableHead>
                  <TableHead>启用</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rules.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell>{r.id}</TableCell>
                    <TableCell className="font-medium">{r.name}</TableCell>
                    <TableCell>{r.path_prefix}</TableCell>
                    <TableCell>{r.methods}</TableCell>
                    <TableCell>
                      <Badge variant={r.rule_type === "duplicate_ip" ? "default" : "secondary"}>
                        {r.rule_type === "duplicate_ip" ? "重复 IP" : "速率限制"}
                      </Badge>
                    </TableCell>
                    <TableCell>{r.max_attempts}</TableCell>
                    <TableCell>{r.enabled ? "是" : "否"}</TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button variant="ghost" size="icon" onClick={() => openEdit(r)}>
                          <Edit className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => handleDelete(r.id)}>
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {rules.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={8} className="py-8 text-center text-muted-foreground">
                      暂无规则
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{editing ? "编辑规则" : "添加规则"}</DialogTitle>
            <DialogDescription>配置风控匹配条件与拦截响应</DialogDescription>
          </DialogHeader>
          {!editing && (
            <div className="space-y-2 pt-2">
              <Label className="text-muted-foreground">快速模板</Label>
              <div className="flex flex-wrap gap-2">
                {ruleTemplates.map((t) => (
                  <Button
                    key={t.label}
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={() => setForm({ ...t.form, domain_id: domainId })}
                  >
                    {t.label}
                  </Button>
                ))}
              </div>
            </div>
          )}

          <div className="grid gap-5 py-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="name">规则名称</Label>
              <Input
                id="name"
                placeholder="例如：注册防重复"
                value={form.name || ""}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="path_prefix">路径前缀</Label>
              <Input
                id="path_prefix"
                placeholder="例如：/api/register"
                value={form.path_prefix || ""}
                onChange={(e) => setForm({ ...form, path_prefix: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="methods">HTTP 方法</Label>
              <Input
                id="methods"
                placeholder="POST,PUT 或 ALL"
                value={form.methods || ""}
                onChange={(e) => setForm({ ...form, methods: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>规则类型</Label>
              <Select value={form.rule_type} onValueChange={(v) => setForm({ ...form, rule_type: v as Rule["rule_type"] })}>
                <SelectTrigger>
                  <SelectValue placeholder="选择规则类型" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="duplicate_ip">重复 IP</SelectItem>
                  <SelectItem value="rate_limit">速率限制</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="identity_fields">身份字段</Label>
              <Input
                id="identity_fields"
                placeholder="phone,email"
                value={form.identity_fields || ""}
                onChange={(e) => setForm({ ...form, identity_fields: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="max_attempts">最大次数</Label>
              <Input
                id="max_attempts"
                type="number"
                placeholder="1"
                value={form.max_attempts || ""}
                onChange={(e) => setForm({ ...form, max_attempts: Number(e.target.value) })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="window_seconds">窗口（秒，0=永久）</Label>
              <Input
                id="window_seconds"
                type="number"
                placeholder="0"
                value={form.window_seconds || ""}
                onChange={(e) => setForm({ ...form, window_seconds: Number(e.target.value) })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="block_status">拦截状态码</Label>
              <Input
                id="block_status"
                type="number"
                placeholder="403"
                value={form.block_status || ""}
                onChange={(e) => setForm({ ...form, block_status: Number(e.target.value) })}
              />
            </div>
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="block_response">拦截响应体（JSON）</Label>
              <Input
                id="block_response"
                placeholder='{"code":403,"message":"重复注册"}'
                value={form.block_response || ""}
                onChange={(e) => setForm({ ...form, block_response: e.target.value })}
              />
              <p className="text-xs text-muted-foreground">
                浏览器访问时会渲染为炫酷防火墙警告页。可配置 {"title"}、{"message"}、{"detail"} 字段自定义页面文案，API 请求仍返回 JSON。
              </p>
            </div>
            <div className="flex items-center gap-2 md:col-span-2">
              <Checkbox id="enabled" checked={form.enabled} onCheckedChange={(v) => setForm({ ...form, enabled: v === true })} />
              <Label htmlFor="enabled" className="cursor-pointer">启用该规则</Label>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>取消</Button>
            <Button onClick={handleSave}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

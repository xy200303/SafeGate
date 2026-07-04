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
  query_match: "",
  methods: "POST",
  rule_type: "duplicate_ip",
  identity_fields: "",
  success_statuses: "2xx",
  success_location_match: "",
  failure_location_match: "",
  max_attempts: 1,
  window_seconds: 0,
  block_seconds: 0,
  block_status: 403,
  block_response: '{"code":403,"message":"重复注册"}',
  enabled: true,
}

const duplicateBlockResponse = '{"code":403,"message":"重复注册"}'
const rateLimitBlockResponse = '{"code":429,"message":"请求过于频繁"}'

type SelectOption = {
  value: string
  label: string
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
    label: "index.php 注册表单",
    form: {
      ...emptyRule,
      name: "注册表单防重复",
      path_prefix: "/index.php",
      query_match: "e=index.post_register",
      identity_fields: "mobile,accountname,bankaccount",
      success_statuses: "2xx,302",
      failure_location_match: "key=username_repeat_register",
    },
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
      block_status: 429,
      block_response: '{"code":429,"message":"请求过于频繁"}',
    },
  },
]

const methodOptions = [
  { value: "POST", label: "POST" },
  { value: "GET", label: "GET" },
  { value: "PUT", label: "PUT" },
  { value: "PATCH", label: "PATCH" },
  { value: "DELETE", label: "DELETE" },
  { value: "OPTIONS", label: "OPTIONS" },
  { value: "HEAD", label: "HEAD" },
  { value: "GET,POST", label: "GET,POST" },
  { value: "POST,PUT", label: "POST,PUT" },
  { value: "ALL", label: "ALL（全部）" },
]

const successStatusOptions = [
  { value: "2xx", label: "2xx（所有成功响应）" },
  { value: "2xx,302", label: "2xx,302（成功响应或跳转）" },
  { value: "302", label: "302（只按跳转判断）" },
  { value: "200-299", label: "200-299（成功范围）" },
  { value: "200,201,204", label: "200,201,204（常见 API 成功）" },
]

const duplicateAttemptOptions = [
  { value: "1", label: "1 次（只允许一次成功）" },
  { value: "2", label: "2 次" },
  { value: "3", label: "3 次" },
  { value: "5", label: "5 次" },
]

const rateLimitAttemptOptions = [
  { value: "5", label: "5 次" },
  { value: "10", label: "10 次" },
  { value: "30", label: "30 次" },
  { value: "60", label: "60 次" },
  { value: "100", label: "100 次" },
]

const duplicateWindowOptions = [
  { value: "0", label: "0（永久）" },
  { value: "3600", label: "1 小时" },
  { value: "86400", label: "1 天" },
  { value: "604800", label: "7 天" },
]

const rateLimitWindowOptions = [
  { value: "10", label: "10 秒" },
  { value: "60", label: "1 分钟" },
  { value: "300", label: "5 分钟" },
  { value: "3600", label: "1 小时" },
]

const duplicateBlockStatusOptions = [
  { value: "403", label: "403 Forbidden" },
  { value: "409", label: "409 Conflict" },
  { value: "400", label: "400 Bad Request" },
  { value: "429", label: "429 Too Many Requests" },
]

const rateLimitBlockStatusOptions = [
  { value: "429", label: "429 Too Many Requests" },
  { value: "403", label: "403 Forbidden" },
  { value: "503", label: "503 Service Unavailable" },
]

const duplicateIdentityOptions = [
  { value: "__ip_only__", label: "只按 IP" },
  { value: "mobile", label: "手机号 mobile" },
  { value: "phone", label: "手机号 phone" },
  { value: "username,mobile,email,bankaccount", label: "用户名 + 手机 + 邮箱 + 银行账号" },
  { value: "mobile,accountname,bankaccount", label: "手机 + 账户名 + 收款账号" },
  { value: "user.phone,user.email", label: "嵌套表单 user.phone + user.email" },
]

const rateLimitIdentityOptions = [
  { value: "__ip_only__", label: "只按 IP" },
  { value: "user_id", label: "用户 ID user_id" },
  { value: "username", label: "用户名 username" },
  { value: "mobile", label: "手机号 mobile" },
]

const duplicateBlockResponseOptions = [
  { value: duplicateBlockResponse, label: "重复注册" },
  { value: '{"code":403,"message":"请勿重复提交"}', label: "重复提交" },
  { value: '{"code":409,"message":"该信息已提交过"}', label: "信息已存在" },
]

const rateLimitBlockResponseOptions = [
  { value: rateLimitBlockResponse, label: "请求过于频繁" },
  { value: '{"code":429,"message":"操作太快，请稍后再试"}', label: "操作太快" },
  { value: '{"code":503,"message":"服务繁忙，请稍后再试"}', label: "服务繁忙" },
]

const withCurrentOption = (options: SelectOption[], value: string, label = value) => {
  if (!value || options.some((item) => item.value === value)) {
    return options
  }
  return [{ value, label }, ...options]
}

const splitPathQuery = (value: string) => {
  const raw = value.trim()
  if (!raw) {
    return { path_prefix: "", query_match: "" }
  }

  try {
    const parsed = new URL(raw, "http://safegate.local")
    return {
      path_prefix: parsed.pathname || "/",
      query_match: parsed.search.replace(/^\?/, ""),
    }
  } catch {
    const [path, query = ""] = raw.split("?", 2)
    return {
      path_prefix: path || "/",
      query_match: query.replace(/^\?/, ""),
    }
  }
}

const joinPathQuery = (pathPrefix?: string, queryMatch?: string) => {
  const path = pathPrefix || ""
  const query = (queryMatch || "").replace(/^\?/, "")
  return query ? `${path}?${query}` : path
}

const stringifyBlockResponse = (value: unknown) => {
  if (typeof value === "string") {
    return value
  }
  if (value == null) {
    return ""
  }
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

export function RulesPage() {
  const [searchParams] = useSearchParams()
  const domainId = Number(searchParams.get("domain_id") || 0)
  const [rules, setRules] = useState<Rule[]>([])
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Rule | null>(null)
  const [form, setForm] = useState<Partial<Rule>>(emptyRule)
  const ruleType = form.rule_type || "duplicate_ip"
  const isDuplicateRule = ruleType === "duplicate_ip"
  const methodValue = (form.methods || "POST").toUpperCase()
  const maxAttemptsValue = String(form.max_attempts ?? (isDuplicateRule ? 1 : 10))
  const windowSecondsValue = String(form.window_seconds ?? (isDuplicateRule ? 0 : 60))
  const blockStatusValue = String(form.block_status ?? (isDuplicateRule ? 403 : 429))
  const successStatusesValue = form.success_statuses || "2xx"
  const identityOptions = isDuplicateRule ? duplicateIdentityOptions : rateLimitIdentityOptions
  const identityPresetValue =
    identityOptions.find((item) => (item.value === "__ip_only__" ? !form.identity_fields : item.value === form.identity_fields))?.value
  const blockResponseOptions = isDuplicateRule ? duplicateBlockResponseOptions : rateLimitBlockResponseOptions
  const blockResponseValue = stringifyBlockResponse(form.block_response)
  const blockResponsePresetValue = blockResponseOptions.find((item) => item.value === blockResponseValue)?.value
  const currentMethodOptions = withCurrentOption(methodOptions, methodValue)
  const currentMaxAttemptOptions = withCurrentOption(isDuplicateRule ? duplicateAttemptOptions : rateLimitAttemptOptions, maxAttemptsValue, `${maxAttemptsValue} 次`)
  const currentWindowOptions = withCurrentOption(isDuplicateRule ? duplicateWindowOptions : rateLimitWindowOptions, windowSecondsValue, `${windowSecondsValue} 秒`)
  const currentBlockStatusOptions = withCurrentOption(isDuplicateRule ? duplicateBlockStatusOptions : rateLimitBlockStatusOptions, blockStatusValue)
  const currentSuccessStatusOptions = withCurrentOption(successStatusOptions, successStatusesValue)

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
    setForm({ ...r, block_response: stringifyBlockResponse(r.block_response) })
    setOpen(true)
  }

  const normalizeJSON = (value: unknown) => {
    if (typeof value !== "string") {
      return value ?? {}
    }
    try {
      return value ? JSON.parse(value) : {}
    } catch {
      return value
    }
  }

  const handleRuleTypeChange = (value: Rule["rule_type"]) => {
    const currentBlockResponse = stringifyBlockResponse(form.block_response)
    const next: Partial<Rule> = { ...form, rule_type: value }

    if (value === "rate_limit") {
      if (!form.window_seconds) next.window_seconds = 60
      if (!form.max_attempts || form.max_attempts === 1) next.max_attempts = 10
      if (!form.block_status || form.block_status === 403) next.block_status = 429
      if (!currentBlockResponse || currentBlockResponse === duplicateBlockResponse) {
        next.block_response = rateLimitBlockResponse
      }
    } else {
      if (form.window_seconds == null) next.window_seconds = 0
      if (!form.max_attempts) next.max_attempts = 1
      if (!form.block_status || form.block_status === 429) next.block_status = 403
      if (!currentBlockResponse || currentBlockResponse === rateLimitBlockResponse) {
        next.block_response = duplicateBlockResponse
      }
      if (!form.success_statuses) next.success_statuses = "2xx"
    }

    setForm(next)
  }

  const handleSave = async () => {
    try {
      const payload = {
        ...form,
        block_response: normalizeJSON(form.block_response),
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
                  <TableHead>路径匹配</TableHead>
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
                    <TableCell>{joinPathQuery(r.path_prefix, r.query_match)}</TableCell>
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
        <DialogContent className="max-h-[90dvh] max-w-2xl grid-rows-[auto_minmax(0,1fr)_auto] gap-0 overflow-hidden p-0">
          <DialogHeader className="px-6 pb-4 pr-12 pt-6">
            <DialogTitle>{editing ? "编辑规则" : "添加规则"}</DialogTitle>
            <DialogDescription>配置风控匹配条件与拦截响应</DialogDescription>
          </DialogHeader>

          <div className="min-h-0 overflow-y-auto px-6">
            {!editing && (
              <div className="space-y-2 pb-4">
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

            <div className="grid gap-5 pb-6 md:grid-cols-2">
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
                <Label htmlFor="path_match">路径 / Query 匹配</Label>
                <Input
                  id="path_match"
                  placeholder="例如：/index.php?e=index.post_register"
                  value={joinPathQuery(form.path_prefix, form.query_match)}
                  onChange={(e) => setForm({ ...form, ...splitPathQuery(e.target.value) })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="methods">HTTP 方法</Label>
                <Select value={methodValue} onValueChange={(v) => setForm({ ...form, methods: v })}>
                  <SelectTrigger id="methods">
                    <SelectValue placeholder="选择 HTTP 方法" />
                  </SelectTrigger>
                  <SelectContent>
                    {currentMethodOptions.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>规则类型</Label>
                <Select value={ruleType} onValueChange={(v) => handleRuleTypeChange(v as Rule["rule_type"])}>
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
                <Select
                  value={identityPresetValue}
                  onValueChange={(v) => setForm({ ...form, identity_fields: v === "__ip_only__" ? "" : v })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择常用身份字段" />
                  </SelectTrigger>
                  <SelectContent>
                    {identityOptions.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Input
                  id="identity_fields"
                  placeholder="phone,email 或 user.phone"
                  value={form.identity_fields || ""}
                  onChange={(e) => setForm({ ...form, identity_fields: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="max_attempts">{isDuplicateRule ? "最大成功次数" : "窗口内最大请求数"}</Label>
                <Select value={maxAttemptsValue} onValueChange={(v) => setForm({ ...form, max_attempts: Number(v) })}>
                  <SelectTrigger id="max_attempts">
                    <SelectValue placeholder="选择次数" />
                  </SelectTrigger>
                  <SelectContent>
                    {currentMaxAttemptOptions.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              {isDuplicateRule && (
                <>
                  <div className="space-y-2">
                    <Label htmlFor="success_statuses">成功状态码</Label>
                    <Select value={successStatusesValue} onValueChange={(v) => setForm({ ...form, success_statuses: v })}>
                      <SelectTrigger id="success_statuses">
                        <SelectValue placeholder="选择成功状态码" />
                      </SelectTrigger>
                      <SelectContent>
                        {currentSuccessStatusOptions.map((item) => (
                          <SelectItem key={item.value} value={item.value}>
                            {item.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="success_location_match">成功 Location 匹配</Label>
                    <Input
                      id="success_location_match"
                      placeholder="例如：key=register_success"
                      value={form.success_location_match || ""}
                      onChange={(e) => setForm({ ...form, success_location_match: e.target.value })}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="failure_location_match">失败 Location 匹配</Label>
                    <Input
                      id="failure_location_match"
                      placeholder="例如：key=username_repeat_register"
                      value={form.failure_location_match || ""}
                      onChange={(e) => setForm({ ...form, failure_location_match: e.target.value })}
                    />
                  </div>
                </>
              )}
              <div className="space-y-2">
                <Label htmlFor="window_seconds">{isDuplicateRule ? "成功计数窗口（秒，0=永久）" : "限流窗口（秒）"}</Label>
                <Select value={windowSecondsValue} onValueChange={(v) => setForm({ ...form, window_seconds: Number(v) })}>
                  <SelectTrigger id="window_seconds">
                    <SelectValue placeholder="选择窗口" />
                  </SelectTrigger>
                  <SelectContent>
                    {currentWindowOptions.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="block_status">拦截状态码</Label>
                <Select value={blockStatusValue} onValueChange={(v) => setForm({ ...form, block_status: Number(v) })}>
                  <SelectTrigger id="block_status">
                    <SelectValue placeholder="选择状态码" />
                  </SelectTrigger>
                  <SelectContent>
                    {currentBlockStatusOptions.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2 md:col-span-2">
                <Label htmlFor="block_response">拦截响应体（JSON）</Label>
                <Select value={blockResponsePresetValue} onValueChange={(v) => setForm({ ...form, block_response: v })}>
                  <SelectTrigger>
                    <SelectValue placeholder="选择拦截响应模板" />
                  </SelectTrigger>
                  <SelectContent>
                    {blockResponseOptions.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Input
                  id="block_response"
                  placeholder='{"code":403,"message":"重复注册"}'
                  value={blockResponseValue}
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
          </div>

          <DialogFooter className="border-t bg-background px-6 py-4">
            <Button variant="outline" onClick={() => setOpen(false)}>取消</Button>
            <Button onClick={handleSave}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

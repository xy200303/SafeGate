import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Edit, Plus, Trash2, Shield } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { type Domain, listDomains, createDomain, updateDomain, deleteDomain } from "@/api/admin"

const emptyDomain: Partial<Domain> = {
  bind_domain: "",
  target_url: "",
  real_ip_headers: "X-Real-IP,X-Forwarded-For,CF-Connecting-IP",
  forward_ip_header: "X-Forwarded-For",
  request_transform: "[]",
  response_transform: "[]",
  rewrite_host: true,
  rewrite_mode: "full",
  is_default: false,
}

function Field({
  label,
  id,
  value,
  onChange,
  type = "text",
  placeholder,
}: {
  label: string
  id: string
  value: string
  onChange: (value: string) => void
  type?: string
  placeholder?: string
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type={type}
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  )
}

export function DomainsPage() {
  const navigate = useNavigate()
  const [domains, setDomains] = useState<Domain[]>([])
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Domain | null>(null)
  const [form, setForm] = useState<Partial<Domain>>(emptyDomain)

  const fetch = async () => {
    setLoading(true)
    try {
      const res = await listDomains()
      setDomains(res.data.data || [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetch()
  }, [])

  const openCreate = () => {
    setEditing(null)
    setForm(emptyDomain)
    setOpen(true)
  }

  const openEdit = (d: Domain) => {
    setEditing(d)
    setForm({ ...d })
    setOpen(true)
  }

  const normalizeJSON = (value: string) => {
    try {
      return value ? JSON.parse(value) : []
    } catch {
      return value
    }
  }

  const handleSave = async () => {
    try {
      const payload = {
        ...form,
        request_transform: normalizeJSON(form.request_transform || ""),
        response_transform: normalizeJSON(form.response_transform || ""),
      }
      if (editing) {
        await updateDomain(editing.id, payload)
      } else {
        await createDomain(payload)
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
      await deleteDomain(id)
      fetch()
    } catch (err: any) {
      alert(err.response?.data?.message || "删除失败")
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">域名映射</h1>
          <p className="text-sm text-muted-foreground">管理绑定域名与目标站点的映射关系</p>
        </div>
        <Button onClick={openCreate}>
          <Plus className="mr-2 h-4 w-4" />
          添加映射
        </Button>
      </div>

      <Card className="border-border/60 shadow-sm">
        <CardHeader>
          <CardTitle>全部映射</CardTitle>
          <CardDescription>所有已配置的域名转发规则</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="py-8 text-center text-muted-foreground">加载中...</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>绑定域名</TableHead>
                  <TableHead>目标地址</TableHead>
                  <TableHead>转发头</TableHead>
                  <TableHead>改写模式</TableHead>
                  <TableHead>默认</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {domains.map((d) => (
                  <TableRow key={d.id}>
                    <TableCell>{d.id}</TableCell>
                    <TableCell className="font-medium">{d.bind_domain}</TableCell>
                    <TableCell className="max-w-xs truncate text-muted-foreground">{d.target_url}</TableCell>
                    <TableCell>{d.forward_ip_header}</TableCell>
                    <TableCell>
                      {d.rewrite_host ? "Host+" : "Host-"}
                      {d.rewrite_mode === "full" ? "完整改写" : d.rewrite_mode === "headers" ? "响应头" : "无改写"}
                    </TableCell>
                    <TableCell>{d.is_default ? <span className="rounded bg-primary/10 px-2 py-0.5 text-xs text-primary">默认</span> : "—"}</TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button variant="outline" size="sm" onClick={() => navigate(`/admin/rules?domain_id=${d.id}`)}>
                          <Shield className="mr-1 h-3 w-3" />
                          规则
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => openEdit(d)}>
                          <Edit className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => handleDelete(d.id)}>
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {domains.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={7} className="py-8 text-center text-muted-foreground">
                      暂无映射，点击右上角添加
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
            <DialogTitle>{editing ? "编辑映射" : "添加映射"}</DialogTitle>
            <DialogDescription>配置域名转发与真实 IP 透传规则</DialogDescription>
          </DialogHeader>
          <div className="grid gap-5 py-4 md:grid-cols-2">
            <Field
              label="绑定域名"
              id="bind_domain"
              value={form.bind_domain || ""}
              onChange={(v) => setForm({ ...form, bind_domain: v })}
              placeholder="例如：api.example.com"
            />
            <Field
              label="目标地址"
              id="target_url"
              value={form.target_url || ""}
              onChange={(v) => setForm({ ...form, target_url: v })}
              placeholder="例如：http://upstream:8080"
            />
            <Field
              label="真实 IP 头"
              id="real_ip_headers"
              value={form.real_ip_headers || ""}
              onChange={(v) => setForm({ ...form, real_ip_headers: v })}
              placeholder="X-Real-IP,X-Forwarded-For"
            />
            <Field
              label="转发 IP 头"
              id="forward_ip_header"
              value={form.forward_ip_header || ""}
              onChange={(v) => setForm({ ...form, forward_ip_header: v })}
              placeholder="X-Forwarded-For"
            />
            <div className="flex items-center gap-2 md:col-span-2">
              <input
                id="rewrite_host"
                type="checkbox"
                checked={form.rewrite_host ?? true}
                onChange={(e) => setForm({ ...form, rewrite_host: e.target.checked })}
                className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
              />
              <Label htmlFor="rewrite_host" className="cursor-pointer">
                改写 Host 头为目标域名（默认开启；关闭时上游收到代理域名）
              </Label>
            </div>
            <div className="flex items-center gap-2 md:col-span-2">
              <input
                id="is_default"
                type="checkbox"
                checked={form.is_default ?? false}
                onChange={(e) => setForm({ ...form, is_default: e.target.checked })}
                className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
              />
              <Label htmlFor="is_default" className="cursor-pointer">
                设为默认站点（当 Host 未匹配任何域名时，请求会落到该站点）
              </Label>
            </div>
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="rewrite_mode">响应改写模式</Label>
              <select
                id="rewrite_mode"
                value={form.rewrite_mode || "full"}
                onChange={(e) => setForm({ ...form, rewrite_mode: e.target.value })}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                <option value="none">仅 Host 改写（不改响应）</option>
                <option value="headers">改写响应头（Location / Set-Cookie）</option>
                <option value="full">完整响应改写（含 HTML body）</option>
              </select>
              <p className="text-xs text-muted-foreground">
                内部/API 站点建议选"仅 Host 改写"；外部公开站点建议选"完整响应改写"。
              </p>
            </div>
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="request_transform">请求体字段映射（JSON 数组）</Label>
              <Input
                id="request_transform"
                placeholder='例如：[{"src":"mobile","dst":"phone"}]'
                value={form.request_transform || ""}
                onChange={(e) => setForm({ ...form, request_transform: e.target.value })}
              />
            </div>
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="response_transform">响应体字段映射（JSON 数组，预留）</Label>
              <Input
                id="response_transform"
                placeholder='例如：[{"src":"phone","dst":"mobile"}]'
                value={form.response_transform || ""}
                onChange={(e) => setForm({ ...form, response_transform: e.target.value })}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSave}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

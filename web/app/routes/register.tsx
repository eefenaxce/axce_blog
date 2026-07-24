import { useState } from "react"
import { useNavigate, Link } from "react-router"
import { Button } from "../components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { api } from "../lib/api"
import { AnimatedPage, FadeIn } from "../components/animate"
import { ArrowLeft, ArrowRight, Check, Mail, User, Lock } from "lucide-react"
import { toast } from "sonner"

const steps = [
  { title: "基本信息", icon: User },
  { title: "邮箱验证", icon: Mail },
  { title: "设置密码", icon: Lock },
]

export default function Register() {
  const [step, setStep] = useState(0)
  const [username, setUsername] = useState("")
  const [nickname, setNickname] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [passwordConfirm, setPasswordConfirm] = useState("")
  const [code, setCode] = useState("")
  const [codeSent, setCodeSent] = useState(false)
  const [codeSending, setCodeSending] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const navigate = useNavigate()

  const sendCode = async () => {
    if (!email) {
      setError("请先填写邮箱")
      return
    }
    setCodeSending(true)
    setError("")
    try {
      await api.sendRegisterCode(email)
      setCodeSent(true)
      toast.success("验证码已发送到您的邮箱")
    } catch (err) {
      const msg = err instanceof Error ? err.message : "发送失败，请稍后重试"
      setError(msg)
      toast.error(msg)
    } finally {
      setCodeSending(false)
    }
  }

  const handleSubmit = async () => {
    if (password !== passwordConfirm) {
      setError("两次输入的密码不一致")
      return
    }
    setLoading(true)
    setError("")
    try {
      const res = await api.register(username, nickname, email, password, passwordConfirm, code)
      toast.success("注册成功！")
      api.setToken(res.data.token)
      // 根据用户组跳转
      if (res.data.user.group === "admin") {
        navigate("/admin")
      } else {
        navigate("/")
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : "注册失败，请稍后重试"
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  const canNext = (): boolean => {
    switch (step) {
      case 0: return username.length >= 3
      case 1: return !!email && !!code && codeSent
      case 2: return password.length >= 6 && password === passwordConfirm
      default: return false
    }
  }

  const nextStep = () => { setError(""); if (step < 2) setStep(step + 1) }
  const prevStep = () => { setError(""); if (step > 0) setStep(step - 1) }

  return (
    <div className="min-h-screen flex items-center justify-center bg-muted/50">
      <AnimatedPage className="w-full max-w-md">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="text-2xl text-center">创建账号</CardTitle>
            <div className="flex items-center justify-center gap-2 mt-4">
              {steps.map((s, i) => (
                <div key={i} className="flex items-center gap-2">
                  <div
                    className={`flex items-center gap-1.5 px-3 py-2.5 rounded-full text-xs font-medium transition-all duration-300 ${
                      i === step
                        ? "bg-primary text-primary-foreground scale-105"
                        : i < step
                        ? "bg-primary/10 text-primary"
                        : "bg-muted text-muted-foreground"
                    }`}
                  >
                    <s.icon className="h-3 w-3" />
                    <span className="hidden sm:inline">{s.title}</span>
                  </div>
                  {i < steps.length - 1 && (
                    <div className={`h-px w-6 transition-colors duration-300 ${i < step ? "bg-primary" : "bg-muted"}`} />
                  )}
                </div>
              ))}
            </div>
          </CardHeader>
          <CardContent>
            {error && (
              <FadeIn>
                <div className="bg-destructive/10 text-destructive text-sm p-3 rounded-md mb-4">{error}</div>
              </FadeIn>
            )}

            {step === 0 && (
              <FadeIn key="step0">
                <div className="space-y-4">
                  <p className="text-sm text-muted-foreground mb-4">请输入您的基本信息</p>
                  <div className="space-y-2">
                    <Label htmlFor="username">用户名</Label>
                    <Input id="username" value={username} onChange={(e) => setUsername(e.target.value)} required minLength={3} placeholder="至少3个字符" />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="nickname">昵称</Label>
                    <Input id="nickname" value={nickname} onChange={(e) => setNickname(e.target.value)} placeholder="可选" />
                  </div>
                  <Button onClick={nextStep} disabled={!canNext()} className="w-full">
                    下一步 <ArrowRight className="h-4 w-4 ml-2" />
                  </Button>
                </div>
              </FadeIn>
            )}

            {step === 1 && (
              <FadeIn key="step1">
                <div className="space-y-4">
                  <p className="text-sm text-muted-foreground mb-4">验证您的邮箱地址</p>
                  <div className="space-y-2">
                    <Label htmlFor="email">邮箱</Label>
                    <div className="flex gap-2">
                      <Input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required className="flex-1" placeholder="your@email.com" />
                      <Button type="button" variant="outline" onClick={sendCode} disabled={codeSending || codeSent || !email} className="whitespace-nowrap">
                        {codeSending ? "发送中..." : codeSent ? "已发送" : "发送验证码"}
                      </Button>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="code">验证码</Label>
                    <Input id="code" value={code} onChange={(e) => setCode(e.target.value)} required placeholder="请输入6位验证码" maxLength={6} />
                  </div>
                  <div className="flex gap-2">
                    <Button variant="outline" onClick={prevStep} className="flex-1"><ArrowLeft className="h-4 w-4 mr-2" />上一步</Button>
                    <Button onClick={nextStep} disabled={!canNext()} className="flex-1">下一步<ArrowRight className="h-4 w-4 ml-2" /></Button>
                  </div>
                </div>
              </FadeIn>
            )}

            {step === 2 && (
              <FadeIn key="step2">
                <div className="space-y-4">
                  <p className="text-sm text-muted-foreground mb-4">设置您的登录密码</p>
                  <div className="space-y-2">
                    <Label htmlFor="password">密码</Label>
                    <Input id="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={6} placeholder="至少6个字符" />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="password_confirm">确认密码</Label>
                    <Input id="password_confirm" type="password" value={passwordConfirm} onChange={(e) => setPasswordConfirm(e.target.value)} required minLength={6} placeholder="再次输入密码" />
                    {passwordConfirm && password !== passwordConfirm && (
                      <p className="text-xs text-destructive">两次输入的密码不一致</p>
                    )}
                  </div>
                  <div className="flex gap-2">
                    <Button variant="outline" onClick={prevStep} className="flex-1"><ArrowLeft className="h-4 w-4 mr-2" />上一步</Button>
                    <Button onClick={handleSubmit} disabled={!canNext() || loading} className="flex-1">
                      {loading ? "注册中..." : "完成注册"}<Check className="h-4 w-4 ml-2" />
                    </Button>
                  </div>
                </div>
              </FadeIn>
            )}

            <div className="mt-4 text-center text-sm text-muted-foreground">
              已有账号？ <Link to="/login" className="text-primary hover:underline">立即登录</Link>
            </div>
          </CardContent>
        </Card>
      </AnimatedPage>
    </div>
  )
}

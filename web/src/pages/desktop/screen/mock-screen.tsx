export function MockScreen() {
  return (
    <div
      className="relative flex h-full w-full select-none items-center justify-center overflow-hidden bg-[#0b1020] text-slate-100"
      data-testid="mock-desktop-screen"
    >
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_20%,rgba(29,78,216,0.32),transparent_34%),radial-gradient(circle_at_80%_75%,rgba(6,182,212,0.22),transparent_38%)]" />
      <div className="absolute left-5 top-4 flex items-center gap-2 text-xs tracking-wide text-slate-400">
        <span className="h-2 w-2 rounded-full bg-emerald-400 shadow-[0_0_10px_#34d399]" />
        MOCK VIDEO INPUT · 1920 × 1080
      </div>
      <div className="relative w-[min(78%,760px)] overflow-hidden rounded-xl border border-white/10 bg-slate-950/75 shadow-2xl backdrop-blur">
        <div className="flex items-center gap-2 border-b border-white/10 bg-white/5 px-4 py-3">
          <span className="h-2.5 w-2.5 rounded-full bg-red-400" />
          <span className="h-2.5 w-2.5 rounded-full bg-amber-300" />
          <span className="h-2.5 w-2.5 rounded-full bg-emerald-400" />
          <span className="ml-3 font-mono text-xs text-slate-400">developer@nanokvm-mock:~</span>
        </div>
        <div className="space-y-2 p-6 font-mono text-[clamp(11px,1.4vw,16px)] leading-relaxed">
          <p className="text-cyan-300">NanoKVM Development Host</p>
          <p><span className="text-emerald-400">✓</span> HDMI capture connected</p>
          <p><span className="text-emerald-400">✓</span> USB HID ready</p>
          <p><span className="text-emerald-400">✓</span> Virtual media mounted</p>
          <p className="pt-3 text-slate-400">Use the toolbar to exercise controls. No hardware is required.</p>
          <p><span className="text-cyan-300">developer@nanokvm-mock</span>:<span className="text-blue-300">~</span>$ <span className="animate-pulse">_</span></p>
        </div>
      </div>
      <div className="absolute bottom-4 right-5 rounded bg-black/35 px-2 py-1 font-mono text-[10px] text-slate-500">
        MSW STATIC FRAME
      </div>
    </div>
  );
}

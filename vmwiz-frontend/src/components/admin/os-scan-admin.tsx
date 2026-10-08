"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
    fetchOsScanInfo,
    fetchSOSHosts,
    previewOsScanMail,
    runOsScan,
    sendOsScanMail,
    updateSOSHosts,
} from "@/lib/api";
import type {
    OsScanMailPreviewResponse,
    OsScanMailResponse,
    OsScanPool,
    OsScanReport,
    OsScanResult,
} from "@/lib/types/api";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import {
    AlertTriangle,
    CheckCircle2,
    ChevronDown,
    ChevronRight,
    Mail,
    Pencil,
    RefreshCw,
    Search,
    Server,
    ShieldCheck,
    XCircle,
} from "lucide-react";
import { cn } from "@/lib/utils";

const STATUS_LABEL: Record<OsScanResult["status"], string> = {
    ok: "OK",
    unreachable: "Unreachable",
    no_ssh_banner: "No SSH banner",
    unknown_os: "Unknown OS",
    unknown_version: "Unknown version",
};

function StatusBadge({ result }: { result: OsScanResult }) {
    if (result.status === "ok" && !result.outdated) {
        return (
            <Badge variant="secondary" className="gap-1">
                <ShieldCheck className="size-3" />
                {result.displayName}
            </Badge>
        );
    }
    if (result.status === "ok" && result.outdated) {
        return (
            <Badge variant="destructive" className="gap-1">
                <AlertTriangle className="size-3" />
                {result.displayName}
            </Badge>
        );
    }
    return (
        <Badge variant="outline" className="gap-1 text-muted-foreground">
            <XCircle className="size-3" />
            {STATUS_LABEL[result.status]}
        </Badge>
    );
}

const DEFAULT_TEMPLATE = `Hi,

Your VM {{hostname}} is running {{display}}.

This OS version is reaching (or has reached) end of support. With the ETH BOT you have to keep your VM up-to-date.
Please upgrade your VM accordingly! If the OS has reached end of life and we don't get a response and it isn't updated in two weeks we will shutdown the VM.
After the VM is shutdown, it will be automatically deleted three weeks later.

Thanks,
VSOS team
`;

const DEFAULT_SUBJECT = `[SOSETH] Please plan an OS upgrade for {{hostname}} ({{display}})`;

export function OsScanAdmin() {
    const [report, setReport] = useState<OsScanReport | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Scan inputs
    const [cidr, setCidr] = useState("");
    const [includeSosHosts, setIncludeSosHosts] = useState(false);
    const [includeProxmox, setIncludeProxmox] = useState(false);
    const [extraHostsText, setExtraHostsText] = useState("");
    const [timeoutSec, setTimeoutSec] = useState(3);

    // Editable SOS list
    const [sosHosts, setSosHosts] = useState<string[]>([]);
    const [editSosOpen, setEditSosOpen] = useState(false);

    // Mail dialog target
    const [mailTarget, setMailTarget] = useState<OsScanPool | null>(null);

    useEffect(() => {
        (async () => {
            try {
                const [info, sos] = await Promise.all([
                    fetchOsScanInfo(),
                    fetchSOSHosts(),
                ]);
                setCidr(info.defaultCidr);
                setSosHosts(sos.hosts);
            } catch (e) {
                setError(e instanceof Error ? e.message : String(e));
            }
        })();
    }, []);

    const runScan = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const cidrRanges = cidr
                .split(/[\s,]+/)
                .map((s) => s.trim())
                .filter(Boolean);
            const extraHosts = extraHostsText
                .split(/[\s,]+/)
                .map((s) => s.trim())
                .filter(Boolean);
            const data = await runOsScan({
                includeProxmox,
                includeSosHosts,
                cidrRanges,
                extraHosts,
                timeoutMs: Math.max(1, timeoutSec) * 1000,
            });
            setReport(data);
        } catch (e) {
            setError(e instanceof Error ? e.message : String(e));
        } finally {
            setLoading(false);
        }
    }, [cidr, includeSosHosts, includeProxmox, extraHostsText, timeoutSec]);

    const stats = useMemo(() => {
        if (!report) return null;
        let ok = 0;
        let outdated = 0;
        let bad = 0;
        for (const r of report.results) {
            if (r.status === "ok" && !r.outdated) ok++;
            else if (r.status === "ok" && r.outdated) outdated++;
            else bad++;
        }
        return { total: report.results.length, ok, outdated, bad };
    }, [report]);

    // Display key for a host: prefer the reverse-DNS name when scanning by IP
    // so pool listings show "files.sos.ethz.ch" instead of "192.33.91.x". When
    // we already scanned by hostname, host==display so this is a no-op.
    const displayByHost = useMemo(() => {
        const m = new Map<string, string>();
        if (!report) return m;
        for (const r of report.results) {
            m.set(r.host, r.rdns && r.rdns !== "" ? r.rdns : r.host);
        }
        return m;
    }, [report]);
    const displayFor = (host: string) => displayByHost.get(host) ?? host;

    return (
        <div className="space-y-4">
            {/* Compact toolbar */}
            <div className="rounded-xl border bg-gradient-to-br from-background to-muted/30 p-3 shadow-sm">
                <div className="flex flex-wrap items-end gap-2">
                    <div className="min-w-[14rem] flex-1 space-y-1">
                        <Label className="text-[11px]">CIDR range(s)</Label>
                        <Input
                            placeholder="192.33.91.0/24"
                            value={cidr}
                            onChange={(e) => setCidr(e.target.value)}
                            className="h-8 font-mono text-xs"
                        />
                    </div>
                    <div className="space-y-1">
                        <Label className="text-[11px]">Timeout</Label>
                        <Input
                            type="number"
                            min={1}
                            max={30}
                            value={timeoutSec}
                            onChange={(e) =>
                                setTimeoutSec(Number(e.target.value) || 3)
                            }
                            className="h-8 w-16"
                        />
                    </div>
                    <Button
                        onClick={runScan}
                        disabled={loading}
                        className="h-8 gap-1.5"
                    >
                        {loading ? (
                            <>
                                <RefreshCw className="size-3.5 animate-spin" />
                                Scanning…
                            </>
                        ) : (
                            <>
                                <Search className="size-3.5" />
                                Scan
                            </>
                        )}
                    </Button>
                </div>

                <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs">
                    <label className="flex items-center gap-1.5">
                        <Checkbox
                            checked={includeSosHosts}
                            onCheckedChange={(v) =>
                                setIncludeSosHosts(Boolean(v))
                            }
                        />
                        <span>
                            SOS machines
                            <span className="text-muted-foreground">
                                {" "}
                                ({sosHosts.length})
                            </span>
                        </span>
                        <Button
                            size="xs"
                            variant="ghost"
                            onClick={() => setEditSosOpen(true)}
                            className="ml-0 size-5 p-0"
                            title="Edit SOS host list"
                        >
                            <Pencil className="size-3" />
                        </Button>
                    </label>
                    <label className="flex items-center gap-1.5">
                        <Checkbox
                            checked={includeProxmox}
                            onCheckedChange={(v) =>
                                setIncludeProxmox(Boolean(v))
                            }
                        />
                        Proxmox cluster VMs
                    </label>
                    <details className="ml-auto cursor-pointer text-muted-foreground">
                        <summary className="text-xs hover:text-foreground">
                            Extra hosts
                        </summary>
                        <Textarea
                            placeholder="example.sos.ethz.ch"
                            value={extraHostsText}
                            onChange={(e) => setExtraHostsText(e.target.value)}
                            className="mt-1 min-h-14 w-72 font-mono text-xs"
                        />
                    </details>
                </div>

                {error && (
                    <div className="mt-2 flex items-start gap-2 rounded-md border border-destructive/30 bg-destructive/10 p-2 text-xs text-destructive">
                        <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
                        <span>{error}</span>
                    </div>
                )}
            </div>

            {report && stats && (
                <>
                    {/* Stats strip */}
                    <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
                        <SummaryCard
                            label="Hosts scanned"
                            value={stats.total}
                            tone="neutral"
                            icon={<Server className="size-3.5" />}
                        />
                        <SummaryCard
                            label="Up to date"
                            value={stats.ok}
                            tone="good"
                            icon={<CheckCircle2 className="size-3.5" />}
                        />
                        <SummaryCard
                            label="Outdated"
                            value={stats.outdated}
                            tone="warn"
                            icon={<AlertTriangle className="size-3.5" />}
                        />
                        <SummaryCard
                            label="Unreachable / unknown"
                            value={stats.bad}
                            tone="bad"
                            icon={<XCircle className="size-3.5" />}
                        />
                    </div>

                    {/* Pools first (full width), then the full host list below it */}
                    <div className="space-y-5">
                        <section className="space-y-2">
                            <h3 className="text-base font-semibold text-foreground">
                                Pools (grouped by OS version)
                            </h3>
                            {report.pools.length === 0 ? (
                                <p className="rounded-md border bg-muted/30 p-3 text-xs text-muted-foreground">
                                    No pools — nothing identifiable was found.
                                </p>
                            ) : (
                                <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
                                    {report.pools.map((pool) => (
                                        <PoolCard
                                            key={pool.poolKey}
                                            pool={pool}
                                            displayFor={displayFor}
                                            onMail={() => setMailTarget(pool)}
                                        />
                                    ))}
                                </div>
                            )}
                        </section>

                        <section className="space-y-2">
                            <h3 className="text-base font-semibold text-foreground">
                                All hosts ({report.results.length})
                            </h3>
                            <div className="max-h-[40rem] overflow-auto rounded-lg border bg-background">
                                <Table>
                                    <TableHeader className="sticky top-0 bg-muted/80 backdrop-blur">
                                        <TableRow>
                                            <TableHead className="text-[11px]">
                                                Host
                                            </TableHead>
                                            <TableHead className="text-[11px]">
                                                OS
                                            </TableHead>
                                            <TableHead className="text-right text-[11px]">
                                                ms
                                            </TableHead>
                                        </TableRow>
                                    </TableHeader>
                                    <TableBody>
                                        {report.results.map((r) => {
                                            const name = displayFor(r.host);
                                            return (
                                                <TableRow key={r.host}>
                                                    <TableCell
                                                        className="font-mono text-xs"
                                                        title={
                                                            r.banner ||
                                                            r.error ||
                                                            ""
                                                        }
                                                    >
                                                        <div className="break-all">
                                                            {name}
                                                        </div>
                                                        {name !== r.host && (
                                                            <div className="break-all text-[10px] text-muted-foreground">
                                                                {r.host}
                                                            </div>
                                                        )}
                                                    </TableCell>
                                                    <TableCell>
                                                        <StatusBadge
                                                            result={r}
                                                        />
                                                    </TableCell>
                                                    <TableCell className="text-right text-[10px] text-muted-foreground">
                                                        {r.durationMs}
                                                    </TableCell>
                                                </TableRow>
                                            );
                                        })}
                                    </TableBody>
                                </Table>
                            </div>
                        </section>
                    </div>
                </>
            )}

            {mailTarget && (
                <MailDialog
                    pool={mailTarget}
                    onClose={() => setMailTarget(null)}
                />
            )}

            {editSosOpen && (
                <SOSHostsDialog
                    initialHosts={sosHosts}
                    onClose={() => setEditSosOpen(false)}
                    onSaved={(hs) => {
                        setSosHosts(hs);
                        setEditSosOpen(false);
                    }}
                />
            )}
        </div>
    );
}

function SummaryCard({
    label,
    value,
    tone,
    icon,
}: {
    label: string;
    value: number;
    tone: "neutral" | "good" | "warn" | "bad";
    icon: React.ReactNode;
}) {
    const toneClasses = {
        neutral: "from-muted/50 to-background ring-foreground/10",
        good: "from-teal-50 to-background ring-teal-300/40 dark:from-teal-950/30",
        warn: "from-amber-50 to-background ring-amber-300/40 dark:from-amber-950/30",
        bad: "from-red-50 to-background ring-red-300/40 dark:from-red-950/30",
    } as const;
    const toneIcon = {
        neutral: "text-muted-foreground",
        good: "text-teal-600",
        warn: "text-amber-600",
        bad: "text-red-600",
    } as const;
    return (
        <div
            className={cn(
                "rounded-lg bg-gradient-to-br p-2.5 ring-1 shadow-sm",
                toneClasses[tone],
            )}
        >
            <div className="flex items-center justify-between text-[11px] text-muted-foreground">
                <span className="truncate">{label}</span>
                <span className={toneIcon[tone]}>{icon}</span>
            </div>
            <div className="mt-0.5 text-xl font-semibold tabular-nums leading-tight">
                {value}
            </div>
        </div>
    );
}

function PoolCard({
    pool,
    displayFor,
    onMail,
}: {
    pool: OsScanPool;
    displayFor: (host: string) => string;
    onMail: () => void;
}) {
    const PREVIEW_COUNT = 5;
    const [expanded, setExpanded] = useState(false);
    // Sort by their human-readable display so the same hostname surfaces
    // regardless of whether we scanned by IP or by name.
    const sortedHosts = useMemo(
        () =>
            [...pool.hosts].sort((a, b) =>
                displayFor(a).localeCompare(displayFor(b)),
            ),
        [pool.hosts, displayFor],
    );
    const visibleHosts = expanded
        ? sortedHosts
        : sortedHosts.slice(0, PREVIEW_COUNT);
    const hiddenCount = sortedHosts.length - visibleHosts.length;
    return (
        <div
            className={cn(
                "group flex flex-col rounded-lg border p-2.5 shadow-sm transition-all hover:shadow-md",
                pool.outdated
                    ? "border-amber-300/60 bg-gradient-to-br from-amber-50/60 to-background dark:from-amber-950/20"
                    : "bg-gradient-to-br from-muted/30 to-background",
            )}
        >
            <div className="flex items-start justify-between gap-2">
                <div className="flex min-w-0 items-center gap-1.5">
                    {pool.outdated ? (
                        <AlertTriangle className="size-3.5 shrink-0 text-amber-600" />
                    ) : (
                        <ShieldCheck className="size-3.5 shrink-0 text-teal-600" />
                    )}
                    <span className="truncate text-sm font-semibold">
                        {pool.displayName}
                    </span>
                    <Badge variant="outline" className="shrink-0">
                        {pool.hosts.length}
                    </Badge>
                </div>
                <Button
                    size="xs"
                    variant={pool.outdated ? "default" : "outline"}
                    onClick={onMail}
                    className="gap-1"
                    title="Mail pool"
                >
                    <Mail className="size-3" />
                    Mail
                </Button>
            </div>
            <ul
                className={cn(
                    "mt-1.5 space-y-0.5 rounded bg-muted/30 px-1.5 py-1 font-mono text-[10.5px]",
                    expanded && "max-h-44 overflow-y-auto",
                )}
            >
                {visibleHosts.map((h) => {
                    const name = displayFor(h);
                    return (
                        <li
                            key={h}
                            className="truncate"
                            title={name === h ? h : `${name} (${h})`}
                        >
                            {name}
                        </li>
                    );
                })}
            </ul>
            {pool.hosts.length > PREVIEW_COUNT && (
                <button
                    type="button"
                    onClick={() => setExpanded((v) => !v)}
                    className="mt-1 flex items-center gap-0.5 text-[11px] text-muted-foreground hover:text-foreground"
                >
                    {expanded ? (
                        <>
                            <ChevronDown className="size-3" />
                            Show less
                        </>
                    ) : (
                        <>
                            <ChevronRight className="size-3" />+{hiddenCount}{" "}
                            more
                        </>
                    )}
                </button>
            )}
        </div>
    );
}

function SOSHostsDialog({
    initialHosts,
    onClose,
    onSaved,
}: {
    initialHosts: string[];
    onClose: () => void;
    onSaved: (hosts: string[]) => void;
}) {
    const [text, setText] = useState(initialHosts.join("\n"));
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const save = async () => {
        setSaving(true);
        setError(null);
        try {
            const hosts = text
                .split(/[\s,]+/)
                .map((s) => s.trim())
                .filter(Boolean);
            const res = await updateSOSHosts(hosts);
            onSaved(res.hosts);
        } catch (e) {
            setError(e instanceof Error ? e.message : String(e));
        } finally {
            setSaving(false);
        }
    };

    const count = useMemo(
        () =>
            text
                .split(/[\s,]+/)
                .map((s) => s.trim())
                .filter(Boolean).length,
        [text],
    );

    return (
        <Dialog open onOpenChange={(o) => (!o ? onClose() : undefined)}>
            <DialogContent className="flex max-h-[85vh] max-w-xl flex-col gap-3 overflow-hidden sm:max-w-xl">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <Pencil className="size-4" />
                        Edit SOS host list
                    </DialogTitle>
                    <DialogDescription>
                        These hostnames are scanned when the &quot;Scrape SOS
                        machines&quot; option is enabled. The list is persisted
                        on the backend.
                    </DialogDescription>
                </DialogHeader>
                <Textarea
                    className="min-h-72 font-mono text-xs"
                    value={text}
                    onChange={(e) => setText(e.target.value)}
                />
                <div className="text-xs text-muted-foreground">
                    {count} host{count === 1 ? "" : "s"}
                </div>
                {error && (
                    <div className="rounded-md border border-destructive/30 bg-destructive/10 p-2 text-xs text-destructive">
                        {error}
                    </div>
                )}
                <DialogFooter>
                    <Button variant="outline" onClick={onClose}>
                        Cancel
                    </Button>
                    <Button onClick={save} disabled={saving}>
                        {saving ? "Saving…" : "Save"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

function MailDialog({
    pool,
    onClose,
}: {
    pool: OsScanPool;
    onClose: () => void;
}) {
    const [subject, setSubject] = useState(DEFAULT_SUBJECT);
    const [body, setBody] = useState(DEFAULT_TEMPLATE);
    const [additionalCcText, setAdditionalCcText] = useState("");
    const [includeNoContact, setIncludeNoContact] = useState(false);
    const [preview, setPreview] = useState<OsScanMailPreviewResponse | null>(
        null,
    );
    const [previewLoading, setPreviewLoading] = useState(false);
    const [sending, setSending] = useState(false);
    const [sendResult, setSendResult] = useState<OsScanMailResponse | null>(
        null,
    );
    const [error, setError] = useState<string | null>(null);

    const additionalCc = useMemo(
        () =>
            additionalCcText
                .split(/[\s,]+/)
                .map((s) => s.trim())
                .filter(Boolean),
        [additionalCcText],
    );

    const buildBody = () => ({
        hosts: pool.hosts,
        subject,
        body,
        additionalCc,
        includeNoContact,
    });

    const doPreview = async () => {
        setPreviewLoading(true);
        setError(null);
        try {
            setPreview(await previewOsScanMail(buildBody()));
        } catch (e) {
            setError(e instanceof Error ? e.message : String(e));
        } finally {
            setPreviewLoading(false);
        }
    };

    const doSend = async () => {
        setSending(true);
        setError(null);
        try {
            setSendResult(await sendOsScanMail(buildBody()));
        } catch (e) {
            setError(e instanceof Error ? e.message : String(e));
        } finally {
            setSending(false);
        }
    };

    return (
        <Dialog open onOpenChange={(o) => (!o ? onClose() : undefined)}>
            <DialogContent className="flex max-h-[85vh] max-w-2xl flex-col gap-3 overflow-hidden sm:max-w-2xl">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <Mail className="size-4" />
                        Mail pool: {pool.displayName}
                    </DialogTitle>
                    <DialogDescription>
                        {pool.hosts.length} VM
                        {pool.hosts.length === 1 ? "" : "s"} in this pool. The
                        mail is rendered per VM with placeholders{" "}
                        <code className="rounded bg-muted px-1 text-[11px]">
                            {"{{hostname}}"}
                        </code>
                        ,{" "}
                        <code className="rounded bg-muted px-1 text-[11px]">
                            {"{{display}}"}
                        </code>
                        ,{" "}
                        <code className="rounded bg-muted px-1 text-[11px]">
                            {"{{os}}"}
                        </code>
                        ,{" "}
                        <code className="rounded bg-muted px-1 text-[11px]">
                            {"{{version}}"}
                        </code>
                        ,{" "}
                        <code className="rounded bg-muted px-1 text-[11px]">
                            {"{{codename}}"}
                        </code>
                        .
                    </DialogDescription>
                </DialogHeader>

                <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
                    <div className="space-y-1.5">
                        <Label className="text-xs">Subject</Label>
                        <Input
                            value={subject}
                            onChange={(e) => setSubject(e.target.value)}
                        />
                    </div>
                    <div className="space-y-1.5">
                        <Label className="text-xs">Body</Label>
                        <Textarea
                            value={body}
                            onChange={(e) => setBody(e.target.value)}
                            className="min-h-48 font-mono text-xs"
                        />
                    </div>
                    <div className="space-y-1.5">
                        <Label className="text-xs">
                            Additional Cc (comma- or space-separated)
                        </Label>
                        <Input
                            placeholder="vm-admins@vsos.ethz.ch"
                            value={additionalCcText}
                            onChange={(e) =>
                                setAdditionalCcText(e.target.value)
                            }
                        />
                    </div>
                    <label className="flex items-center gap-2 text-xs text-muted-foreground">
                        <Checkbox
                            checked={includeNoContact}
                            onCheckedChange={(v) =>
                                setIncludeNoContact(Boolean(v))
                            }
                        />
                        Send even when a VM has no contact in its description
                        (uses additional Cc only)
                    </label>

                    {preview && (
                        <div className="rounded-md border bg-muted/20 p-2">
                            <div className="mb-1 text-xs font-semibold">
                                Preview: {preview.totalMails} mail(s) will be
                                sent
                            </div>
                            <div className="max-h-44 overflow-y-auto text-[11px]">
                                <Table>
                                    <TableHeader>
                                        <TableRow>
                                            <TableHead>Host</TableHead>
                                            <TableHead>Recipients</TableHead>
                                        </TableRow>
                                    </TableHeader>
                                    <TableBody>
                                        {preview.perHost.map((p) => (
                                            <TableRow key={p.host}>
                                                <TableCell className="font-mono">
                                                    {p.host}
                                                </TableCell>
                                                <TableCell className="font-mono">
                                                    {p.skipped ? (
                                                        <span className="text-muted-foreground">
                                                            (skipped:{" "}
                                                            {p.skipReason})
                                                        </span>
                                                    ) : (
                                                        (
                                                            p.recipients ?? []
                                                        ).join(", ")
                                                    )}
                                                </TableCell>
                                            </TableRow>
                                        ))}
                                    </TableBody>
                                </Table>
                            </div>
                        </div>
                    )}

                    {sendResult && (
                        <div className="rounded-md border border-teal-300/50 bg-teal-50/60 p-2 text-xs dark:bg-teal-950/30">
                            <div className="font-semibold text-teal-700 dark:text-teal-300">
                                Sent: {sendResult.sent} · Skipped:{" "}
                                {sendResult.skipped} · Failed:{" "}
                                {sendResult.failed}
                            </div>
                            {sendResult.failed > 0 && (
                                <ul className="mt-1 max-h-32 overflow-y-auto font-mono">
                                    {sendResult.perHost
                                        .filter((p) => p.error)
                                        .map((p) => (
                                            <li
                                                key={p.host}
                                                className="text-destructive"
                                            >
                                                {p.host}: {p.error}
                                            </li>
                                        ))}
                                </ul>
                            )}
                        </div>
                    )}

                    {error && (
                        <div className="flex items-start gap-2 rounded-md border border-destructive/30 bg-destructive/10 p-2 text-xs text-destructive">
                            <AlertTriangle className="mt-0.5 size-4 shrink-0" />
                            <span>{error}</span>
                        </div>
                    )}
                </div>

                <DialogFooter>
                    <Button variant="outline" onClick={onClose}>
                        Close
                    </Button>
                    <Button
                        variant="outline"
                        onClick={doPreview}
                        disabled={previewLoading || sending}
                    >
                        {previewLoading ? "Previewing…" : "Preview recipients"}
                    </Button>
                    <Button
                        onClick={doSend}
                        disabled={sending || sendResult !== null}
                        className="gap-1"
                    >
                        {sending ? (
                            <RefreshCw className="size-4 animate-spin" />
                        ) : (
                            <Mail className="size-4" />
                        )}
                        {sendResult ? "Sent" : "Send"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

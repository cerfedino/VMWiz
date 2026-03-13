"use client";

import React, { useState } from "react";
import { useVMRequestForm } from "@/context/vm-request-form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Slider } from "@/components/ui/slider";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { FetchDialog } from "@/components/fetch-dialog";
import {
    submitVMRequest,
    prepareSubmitVMRequest,
    ValidationError,
} from "@/lib/api";
import { Plus, Minus, RotateCcw } from "lucide-react";

function FieldError({ message }: { message?: string }) {
    if (!message) return null;
    return (
        <p className="animate-in fade-in-0 slide-in-from-top-1 duration-200 text-sm text-destructive">
            {message}
        </p>
    );
}

export function VMRequestForm() {
    const { values, isModified, reset, setValidationErrors, clearErrors } =
        useVMRequestForm();
    const [dialogOpen, setDialogOpen] = useState(false);

    function handleSubmit(e: React.FormEvent) {
        e.preventDefault();
        clearErrors();
        setDialogOpen(true);
    }

    function handleError(err: Error): boolean {
        // if we get a ValidationError, we handle it
        if (err instanceof ValidationError) {
            setValidationErrors(err.errors);
            return true;
        }
        // Otherwise we let the FetchDialog show the error
        return false;
    }

    return (
        <>
            <FetchDialog
                open={dialogOpen}
                onOpenChange={setDialogOpen}
                fetchFn={(onConfirm) =>
                    submitVMRequest(
                        values as unknown as Record<string, unknown>,
                        onConfirm,
                    ).then((data) => ({ data }))
                }
                requestInfo={prepareSubmitVMRequest(
                    values as unknown as Record<string, unknown>,
                )}
                immediate
                title="Submitting Request"
                successDescription="We received your request! You can close this window."
                onError={handleError}
            />

            <form
                onSubmit={handleSubmit}
                className="mx-auto w-full max-w-175 space-y-8 p-6 pb-16"
            >
                <div className="text-center">
                    <h1 className="text-2xl font-bold">VM Request Form</h1>
                    <div className="mt-2 flex justify-center">
                        <Button
                            type="button"
                            variant="ghost"
                            size="icon-xs"
                            className={
                                isModified
                                    ? "opacity-100"
                                    : "pointer-events-none opacity-0"
                            }
                            onClick={reset}
                        >
                            <RotateCcw className="text-destructive" />
                            <span className="sr-only">Reset form</span>
                        </Button>
                    </div>
                </div>

                <GeneralInfoSection />

                <Separator className="opacity-30" />

                <VMSpecSection />

                <Separator className="opacity-30" />

                <SshKeysSection />

                <Separator className="opacity-30" />

                <CommentsAndTermsSection />

                <Button
                    type="submit"
                    className="w-full"
                    size="lg"
                    disabled={dialogOpen}
                >
                    Submit request
                </Button>
            </form>
        </>
    );
}

function GeneralInfoSection() {
    const { values, errors, setField, syncToUrl } = useVMRequestForm();

    return (
        <section className="space-y-4">
            <h2 className="text-lg font-semibold">General Information</h2>

            <div className="space-y-2">
                <Label htmlFor="email">University E-Mail address</Label>
                <Input
                    id="email"
                    type="email"
                    placeholder="you@ethz.ch"
                    value={values.email}
                    onChange={(e) => setField("email", e.target.value)}
                    onBlur={syncToUrl}
                    aria-invalid={!!errors.email}
                />
                <FieldError message={errors.email} />
            </div>

            <div className="space-y-2">
                <Label htmlFor="personalEmail">Non-ETH E-Mail address</Label>
                <Input
                    id="personalEmail"
                    type="email"
                    placeholder="you@example.com"
                    value={values.personalEmail}
                    onChange={(e) => setField("personalEmail", e.target.value)}
                    onBlur={syncToUrl}
                    aria-invalid={!!errors.personalEmail}
                />
                <FieldError message={errors.personalEmail} />
            </div>

            <div className="flex items-center gap-2">
                <Checkbox
                    id="isOrganization"
                    checked={values.isOrganization}
                    onCheckedChange={(checked) => {
                        setField("isOrganization", checked === true);
                        syncToUrl();
                    }}
                />
                <Label htmlFor="isOrganization">
                    Are you requesting this VM on behalf of an organization?
                </Label>
            </div>

            {values.isOrganization && (
                <div className="space-y-2">
                    <Label htmlFor="orgName">Organization Name</Label>
                    <Input
                        id="orgName"
                        placeholder="My Organization"
                        value={values.orgName}
                        onChange={(e) => setField("orgName", e.target.value)}
                        onBlur={syncToUrl}
                        aria-invalid={!!errors.orgName}
                    />
                    <FieldError message={errors.orgName} />
                </div>
            )}
        </section>
    );
}

function VMSpecSection() {
    const { values, errors, allowed, setField, syncToUrl } = useVMRequestForm();

    return (
        <section className="space-y-6">
            <h2 className="text-lg font-semibold">VM Specifications</h2>

            {/* Hostname */}
            <div className="space-y-2">
                <Label htmlFor="hostname">Hostname</Label>
                <div className="flex items-center gap-0">
                    <Input
                        id="hostname"
                        placeholder="my-vm"
                        value={values.hostname}
                        onChange={(e) => setField("hostname", e.target.value)}
                        onBlur={syncToUrl}
                        aria-invalid={!!errors.hostname}
                        className="rounded-r-none"
                    />
                    <span className="flex h-8 items-center rounded-r-lg border border-l-0 border-input bg-muted px-3 text-sm text-muted-foreground">
                        .vsos.ethz.ch
                    </span>
                </div>
                <FieldError message={errors.hostname} />
            </div>

            {/* OS Image */}
            <div className="space-y-2">
                <Label>OS Image</Label>
                <Select
                    value={values.image}
                    onValueChange={(val) => {
                        setField("image", val ?? "");
                        syncToUrl();
                    }}
                >
                    <SelectTrigger className="w-full">
                        <SelectValue placeholder="Select an OS image" />
                    </SelectTrigger>
                    <SelectContent>
                        {allowed.image.map((img) => (
                            <SelectItem key={img} value={img}>
                                {img}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
                <FieldError message={errors.image} />
            </div>

            {/* CPU Cores */}
            <div className="space-y-2">
                <Label>CPU Cores</Label>
                <div className="flex items-center gap-4">
                    <Input
                        type="number"
                        className="w-20"
                        min={allowed.cores.min}
                        max={allowed.cores.max}
                        value={values.cores}
                        onChange={(e) =>
                            setField("cores", Number(e.target.value))
                        }
                        onBlur={syncToUrl}
                        aria-invalid={!!errors.cores}
                    />
                    <Slider
                        className="flex-1 [&_[data-slot=slider-range]]:bg-red-500 [&_[data-slot=slider-thumb]]:border-red-500 [&_[data-slot=slider-thumb]]:ring-red-500/50"
                        min={allowed.cores.min}
                        max={allowed.cores.max}
                        step={1}
                        value={[values.cores]}
                        onValueChange={(val) =>
                            setField("cores", Array.isArray(val) ? val[0] : val)
                        }
                        onValueCommitted={syncToUrl}
                    />
                </div>
                <FieldError message={errors.cores} />
            </div>

            {/* RAM */}
            <div className="space-y-2">
                <Label>RAM (GB)</Label>
                <div className="flex items-center gap-4">
                    <Input
                        type="number"
                        className="w-20"
                        min={allowed.ramGB.min}
                        max={allowed.ramGB.max}
                        value={values.ramGB}
                        onChange={(e) =>
                            setField("ramGB", Number(e.target.value))
                        }
                        onBlur={syncToUrl}
                        aria-invalid={!!errors.ramGB}
                    />
                    <Slider
                        className="flex-1 [&_[data-slot=slider-range]]:bg-orange-500 [&_[data-slot=slider-thumb]]:border-orange-500 [&_[data-slot=slider-thumb]]:ring-orange-500/50"
                        min={allowed.ramGB.min}
                        max={allowed.ramGB.max}
                        step={1}
                        value={[values.ramGB]}
                        onValueChange={(val) =>
                            setField("ramGB", Array.isArray(val) ? val[0] : val)
                        }
                        onValueCommitted={syncToUrl}
                    />
                </div>
                <FieldError message={errors.ramGB} />
            </div>

            {/* Disk Space */}
            <div className="space-y-2">
                <Label>Disk Space (GB)</Label>
                <div className="flex items-center gap-4">
                    <Input
                        type="number"
                        className="w-20"
                        min={allowed.diskGB.min}
                        max={allowed.diskGB.max}
                        value={values.diskGB}
                        onChange={(e) =>
                            setField("diskGB", Number(e.target.value))
                        }
                        onBlur={syncToUrl}
                        aria-invalid={!!errors.diskGB}
                    />
                    <Slider
                        className="flex-1 [&_[data-slot=slider-range]]:bg-green-500 [&_[data-slot=slider-thumb]]:border-green-500 [&_[data-slot=slider-thumb]]:ring-green-500/50"
                        min={allowed.diskGB.min}
                        max={allowed.diskGB.max}
                        step={1}
                        value={[values.diskGB]}
                        onValueChange={(val) =>
                            setField(
                                "diskGB",
                                Array.isArray(val) ? val[0] : val,
                            )
                        }
                        onValueCommitted={syncToUrl}
                    />
                </div>
                <FieldError message={errors.diskGB} />
            </div>
        </section>
    );
}

function SshKeysSection() {
    const { values, errors, addSshKey, removeSshKey, updateSshKey, syncToUrl } =
        useVMRequestForm();

    // e.g. "Please provide at least one valid SSH public key"
    const topLevelError =
        values.sshPubkey.length === 0 && errors.sshPubkey.length > 0
            ? errors.sshPubkey.join("\n")
            : typeof errors.sshPubkey === "string"
              ? errors.sshPubkey
              : "";

    return (
        <section className="space-y-4">
            <div className="flex items-center gap-2">
                <h2 className="text-lg font-semibold">SSH Public Key(s)</h2>
                <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    onClick={() => {
                        addSshKey();
                        syncToUrl();
                    }}
                >
                    <Plus />
                    <span className="sr-only">Add SSH key</span>
                </Button>
            </div>

            <FieldError message={topLevelError} />

            {values.sshPubkey.map((key, index) => (
                <div key={index} className="flex items-start gap-2">
                    <Button
                        type="button"
                        variant="ghost"
                        size="icon-xs"
                        className="mt-1.5"
                        disabled={values.sshPubkey.length <= 1}
                        onClick={() => {
                            removeSshKey(index);
                            syncToUrl();
                        }}
                    >
                        <Minus />
                        <span className="sr-only">Remove SSH key</span>
                    </Button>
                    <div className="flex-1 space-y-1">
                        <Input
                            placeholder="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCq..."
                            value={key}
                            onChange={(e) =>
                                updateSshKey(index, e.target.value)
                            }
                            onBlur={syncToUrl}
                            aria-invalid={!!errors.sshPubkey[index]}
                        />
                        <FieldError message={errors.sshPubkey[index]} />
                    </div>
                </div>
            ))}
        </section>
    );
}

function CommentsAndTermsSection() {
    const { values, errors, setField, syncToUrl } = useVMRequestForm();

    return (
        <section className="space-y-4">
            <div className="space-y-2">
                <Label htmlFor="comments">Comments</Label>
                <Textarea
                    id="comments"
                    placeholder="Do you have any special wishes or requirements?"
                    value={values.comments}
                    onChange={(e) => setField("comments", e.target.value)}
                    onBlur={syncToUrl}
                />
                <FieldError message={errors.explanation} />
            </div>

            <div className="flex gap-2">
                <Checkbox
                    id="accept_terms"
                    checked={values.accept_terms}
                    onCheckedChange={(checked) => {
                        setField("accept_terms", checked === true);
                        syncToUrl();
                    }}
                    aria-invalid={!!errors.accept_terms}
                    className="mt-1 shrink-0"
                />
                <label
                    htmlFor="accept_terms"
                    className="text-sm leading-normal select-none"
                >
                    I have read and understood the{" "}
                    <a
                        href="https://rechtssammlung.sp.ethz.ch/Dokumente/203.21.pdf"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-primary underline underline-offset-2 hover:text-primary/80"
                    >
                        Benutzungsordnung für Informations- und
                        Kommunikationstechnologie an der ETH Zürich (BOT)
                    </a>{" "}
                    <span className="text-muted-foreground">
                        (only accessible from ETH network/VPN)
                    </span>
                </label>
            </div>
            <FieldError message={errors.accept_terms} />
        </section>
    );
}

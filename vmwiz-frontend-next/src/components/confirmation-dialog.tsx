"use client";

import React from "react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

interface ConfirmationDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    title: string;
    description?: string;
    icon?: React.ReactNode;
    iconClassName?: string;
    footer?: React.ReactNode;
    children?: React.ReactNode;
}

export function ConfirmationDialog({
    open,
    onOpenChange,
    title,
    description,
    icon,
    iconClassName,
    footer,
    children,
}: ConfirmationDialogProps) {
    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent showCloseButton={false}>
                <DialogHeader>
                    {icon && (
                        <div
                            className={cn(
                                "mx-auto mb-1 flex h-12 w-12 items-center justify-center rounded-full bg-muted",
                                iconClassName,
                            )}
                        >
                            {icon}
                        </div>
                    )}
                    <DialogTitle className="text-center">{title}</DialogTitle>
                    {description && (
                        <DialogDescription className="text-center text-balance">
                            {description}
                        </DialogDescription>
                    )}
                </DialogHeader>

                {children}

                {footer && <DialogFooter>{footer}</DialogFooter>}
            </DialogContent>
        </Dialog>
    );
}

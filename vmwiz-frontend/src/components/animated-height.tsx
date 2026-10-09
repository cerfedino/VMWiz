"use client";

import { useLayoutEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";

/** Transitions its height whenever the content's height changes*/
export function AnimatedHeight({
    children,
    className,
}: {
    children: React.ReactNode;
    className?: string;
}) {
    const inner = useRef<HTMLDivElement>(null);
    const [height, setHeight] = useState<number>();

    useLayoutEffect(() => {
        const el = inner.current;
        if (!el) return;
        const observer = new ResizeObserver(() => setHeight(el.offsetHeight));
        observer.observe(el);
        return () => observer.disconnect();
    }, []);

    return (
        <div
            className={cn(
                "overflow-clip [overflow-clip-margin:4px] transition-[height] duration-300 ease-out",
                className,
            )}
            style={{ height }}
        >
            <div ref={inner}>{children}</div>
        </div>
    );
}

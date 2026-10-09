"use client";

import { useTheme } from "next-themes";
import { Button } from "@/components/ui/button";
import { Moon, Sun } from "lucide-react";

export function ThemeToggle() {
    const { resolvedTheme, setTheme } = useTheme();
    return (
        <Button
            variant="ghost"
            size="icon-sm"
            onClick={() =>
                setTheme(resolvedTheme === "dark" ? "light" : "dark")
            }
        >
            <Sun className="hidden dark:block" />
            <Moon className="dark:hidden" />
            <span className="sr-only">Toggle theme</span>
        </Button>
    );
}

import Link from "next/link";
import { Globe, KeyRound } from "lucide-react";
import { Button } from "@/components/ui/button";

export function Footer() {
    return (
        <footer className="flex flex-col items-center gap-4 border-t bg-white p-6">
            <span className="text-sm text-muted-foreground">
                Made with ❤️ by VSOS
            </span>
            <div className="flex items-center gap-4">
                <Button
                    variant="ghost"
                    size="icon"
                    nativeButton={false}
                    render={
                        <a
                            href="https://vsos.ethz.ch"
                            target="_blank"
                            rel="noopener noreferrer"
                        />
                    }
                >
                    <Globe className="size-4" />
                    <span className="sr-only">VSOS Website</span>
                </Button>
                <Button
                    variant="ghost"
                    size="icon"
                    nativeButton={false}
                    render={<Link href="/console" />}
                >
                    <KeyRound className="size-4" />
                    <span className="sr-only">Admin Console</span>
                </Button>
            </div>
        </footer>
    );
}

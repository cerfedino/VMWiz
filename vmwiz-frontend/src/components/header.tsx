import Image from "next/image";
import Link from "next/link";
import { ThemeToggle } from "@/components/theme-toggle";

export function Header() {
    return (
        <header className="relative flex flex-col items-center gap-1 py-6">
            <div className="absolute top-4 right-4">
                <ThemeToggle />
            </div>
            <Image
                src="/SOSETH_Logo.svg"
                alt="SOSETH Logo"
                width={160}
                height={80}
                className="h-[8vh] max-w-[20vw] w-auto dark:invert"
                priority
            />
            <Link href="/">
                <h1 className="text-2xl font-bold">VMWiz</h1>
            </Link>
        </header>
    );
}

import Image from "next/image";
import Link from "next/link";

export function Header() {
    return (
        <header className="flex flex-col items-center gap-1 py-6">
            <Image
                src="/SOSETH_Logo.svg"
                alt="SOSETH Logo"
                width={160}
                height={80}
                className="h-[8vh] max-w-[20vw] w-auto"
                priority
            />
            <Link href="/">
                <h1 className="text-2xl font-bold">VMWiz</h1>
            </Link>
        </header>
    );
}

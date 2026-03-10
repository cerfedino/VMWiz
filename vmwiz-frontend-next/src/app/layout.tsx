import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { Header } from "@/components/header";
import { Footer } from "@/components/footer";
import { Separator } from "@/components/ui/separator";

const inter = Inter({
    variable: "--font-inter",
    subsets: ["latin"],
});

export const metadata: Metadata = {
    title: "VMWiz",
    description: "VM Request form page",
};

export default function RootLayout({
    children,
}: Readonly<{
    children: React.ReactNode;
}>) {
    return (
        <html lang="en">
            <body
                className={`${inter.className} ${inter.variable} antialiased`}
            >
                <Header />
                <Separator />
                <main className="min-h-[86vh]">{children}</main>
                <Footer />
            </body>
        </html>
    );
}

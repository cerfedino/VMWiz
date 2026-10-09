import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { Header } from "@/components/header";
import { Footer } from "@/components/footer";
import { Separator } from "@/components/ui/separator";
import { ErrorToasterWrapper } from "@/components/error-toaster-wrapper";
import { ThemeProvider } from "next-themes";

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
        <html lang="en" suppressHydrationWarning>
            <body
                className={`${inter.className} ${inter.variable} antialiased`}
            >
                <ThemeProvider
                    attribute="class"
                    defaultTheme="system"
                    enableSystem
                    disableTransitionOnChange
                >
                    <Header />
                    <Separator />
                    <main className="min-h-[86vh]">{children}</main>
                    <Footer />
                    <ErrorToasterWrapper />
                </ThemeProvider>
            </body>
        </html>
    );
}

import { AuthProvider } from "@/context/auth";

export const metadata = {
    title: "VMWiz - Admin Console",
};

export default function ConsoleLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    return <AuthProvider>{children}</AuthProvider>;
}

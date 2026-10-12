import { AuthProvider } from "@/context/auth";

export const metadata = {
    title: "VMWiz - Admin",
};

export default function DashboardLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    return <AuthProvider>{children}</AuthProvider>;
}

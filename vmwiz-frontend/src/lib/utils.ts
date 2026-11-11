export { cn } from "cn";

export const isInstitutionalEmail = (email: string) =>
    /^[^\s@]+@(?:[^\s@]+\.)*(?:ethz|uzh)\.ch$/i.test(email.trim());

/** Format an ISO date string to a readable format (e.g. "Mar 14, 2025, 16:36"). */
export function formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    const options: Intl.DateTimeFormatOptions = {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        hour12: false,
    };
    return date.toLocaleDateString("en-US", options);
}

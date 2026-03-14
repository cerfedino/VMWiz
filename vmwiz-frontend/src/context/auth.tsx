"use client";

import { getBaseURL } from "@/lib/utils";
import React, {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useMemo,
    useState,
} from "react";

export type AuthStatus = "loading" | "authenticated" | "unauthenticated";

interface AuthContextValue {
    /** Current authentication status */
    status: AuthStatus;

    /** The authenticated user's email, if available */
    email: string | null;

    /** Trigger the login flow (redirect to backend auth start) */
    login: () => void;

    /** Re-check authentication status */
    refresh: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth() {
    const ctx = useContext(AuthContext);
    if (!ctx) {
        throw new Error("useAuth must be used within <AuthProvider>");
    }
    return ctx;
}

function redirectToLogin() {
    window.location.href = `${getBaseURL()}/api/auth/start`;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [status, setStatus] = useState<AuthStatus>("loading");
    const [email, setEmail] = useState<string | null>(null);

    // Counter that triggers a re-check when incremented.
    const [checkTrigger, setCheckTrigger] = useState(0);

    useEffect(() => {
        let cancelled = false;

        fetch(`${getBaseURL()}/api/vmrequest`, {
            method: "GET",
            headers: { "Content-Type": "application/json" },
            credentials: "include",
        })
            .then((response) => {
                if (cancelled) return;

                if (response.ok) {
                    setStatus("authenticated");
                    setEmail(null);
                    return;
                }

                console.error(
                    "Auth check failed with status:",
                    response.status,
                );
                redirectToLogin();
            })
            .catch((err) => {
                if (cancelled) return;
                console.error("Auth check failed:", err);
                redirectToLogin();
            });

        return () => {
            cancelled = true;
        };
    }, [checkTrigger]);

    const login = useCallback(() => {
        redirectToLogin();
    }, []);

    const refresh = useCallback(() => {
        setStatus("loading");
        setCheckTrigger((n) => n + 1);
    }, []);

    const ctx = useMemo<AuthContextValue>(
        () => ({
            status,
            email,
            login,
            refresh,
        }),
        [status, email, login, refresh],
    );

    return <AuthContext.Provider value={ctx}>{children}</AuthContext.Provider>;
}

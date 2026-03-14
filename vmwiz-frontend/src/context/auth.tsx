"use client";

import React, {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useState,
} from "react";

/** User info returned by /api/auth/whoami */
export interface WhoAmI {
    email: string;
    groups: string[];
}

interface AuthContextValue {
    /** user info, null until fetched */
    user: WhoAmI | null;
    loading: boolean;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth() {
    const ctx = useContext(AuthContext);
    if (!ctx) {
        throw new Error("useAuth must be used within <AuthProvider>");
    }
    return ctx;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [user, setUser] = useState<WhoAmI | null>(null);
    const [loading, setLoading] = useState(true);

    const fetchWhoAmI = useCallback(async (): Promise<WhoAmI | null> => {
        setLoading(true);
        try {
            const response = await fetch("/api/auth/whoami", {
                method: "GET",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
            });

            if (!response.ok) {
                setUser(null);
                return null;
            }

            const data = (await response.json()) as WhoAmI;
            setUser(data);
            return data;
        } catch (err) {
            setUser(null);
            throw err;
            return null;
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchWhoAmI();
    }, [fetchWhoAmI]);

    return (
        <AuthContext.Provider value={{ user, loading }}>
            {children}
        </AuthContext.Provider>
    );
}

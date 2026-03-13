"use client";

import React, {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useMemo,
    useRef,
    useState,
} from "react";

import { useSearchParams } from "next/navigation";
import {
    VMRequestFormData,
    VMRequestValidationErrors,
    VMRequestAllowedValues,
    DEFAULT_FORM_VALUES,
    DEFAULT_ALLOWED_VALUES,
    EMPTY_VALIDATION_ERRORS,
} from "@/lib/types/api";
import { fetchVMOptions } from "@/lib/api";

interface VMRequestFormContextValue {
    /** Current form field values */
    values: VMRequestFormData;

    /** Per-field validation errors */
    errors: VMRequestValidationErrors;

    /** Allowed values fetched from the backend */
    allowed: VMRequestAllowedValues;

    /** Update a single field */
    setField: <K extends keyof VMRequestFormData>(
        key: K,
        value: VMRequestFormData[K],
    ) => void;

    /** Update multiple fields at once */
    setFields: (partial: Partial<VMRequestFormData>) => void;

    /** Whether the form has been modified from its initial values */
    isModified: boolean;

    /** Persist current form values to the URL query string. */
    syncToUrl: () => void;

    /** Reset the form */
    reset: () => void;

    /** Add a new empty SSH public key entry */
    addSshKey: () => void;

    /** Remove the SSH key at specific index */
    removeSshKey: (index: number) => void;

    /** Update the SSH key at specific index */
    updateSshKey: (index: number, value: string) => void;

    setValidationErrors: (errors: Partial<VMRequestValidationErrors>) => void;
    clearErrors: () => void;
}

const VMRequestFormContext = createContext<VMRequestFormContextValue | null>(
    null,
);

export function useVMRequestForm() {
    const ctx = useContext(VMRequestFormContext);
    if (!ctx) {
        throw new Error(
            "useVMRequestForm must be used within <VMRequestFormProvider>",
        );
    }
    return ctx;
}

function deepClone<T>(obj: T): T {
    return JSON.parse(JSON.stringify(obj));
}

/** Serialise only the fields that differ from `initial` into URLSearchParams */
function toSearchParams(
    current: VMRequestFormData,
    initial: VMRequestFormData,
): URLSearchParams {
    const params = new URLSearchParams();

    for (const _key of Object.keys(current) as (keyof VMRequestFormData)[]) {
        const cur = current[_key];
        const init = initial[_key];

        if (Array.isArray(cur)) {
            const changed = (cur as string[]).filter(
                (v) => v !== "" && !(init as string[]).includes(v),
            );
            if (changed.length > 0) {
                params.set(_key, changed.join(","));
            }
        } else if (cur !== init) {
            params.set(_key, String(cur));
        }
    }

    return params;
}

/** Restore form values from URLSearchParams, using `base` as the foundation */
function fromSearchParams(
    params: URLSearchParams,
    base: VMRequestFormData,
): VMRequestFormData {
    const result = deepClone(base);

    for (const [key, raw] of params.entries()) {
        if (!(key in result)) continue;
        const k = key as keyof VMRequestFormData;

        if (k === "sshPubkey") {
            const parts = raw.split(",").filter((v) => v !== "");
            result.sshPubkey = parts.length > 0 ? parts : [""];
        } else if (typeof result[k] === "boolean") {
            (result as unknown as Record<string, unknown>)[k] = raw === "true";
        } else if (typeof result[k] === "number") {
            const n = Number(raw);
            if (!isNaN(n)) {
                (result as unknown as Record<string, unknown>)[k] = n;
            }
        } else {
            (result as unknown as Record<string, unknown>)[k] = raw;
        }
    }

    return result;
}

function valuesEqual(a: VMRequestFormData, b: VMRequestFormData): boolean {
    for (const _key of Object.keys(a) as (keyof VMRequestFormData)[]) {
        const va = a[_key];
        const vb = b[_key];

        if (Array.isArray(va)) {
            const arrA = va as string[];
            const arrB = vb as string[];
            if (
                arrA.length !== arrB.length ||
                arrA.some((v, i) => v !== arrB[i])
            ) {
                return false;
            }
        } else if (va !== vb) {
            return false;
        }
    }
    return true;
}

export function VMRequestFormProvider({
    children,
}: {
    children: React.ReactNode;
}) {
    const searchParams = useSearchParams();

    // Initial values
    const [initialValues] = useState<VMRequestFormData>(() =>
        deepClone(DEFAULT_FORM_VALUES),
    );

    const [values, setValues] = useState<VMRequestFormData>(() =>
        fromSearchParams(searchParams, deepClone(DEFAULT_FORM_VALUES)),
    );

    // Always-current ref so syncToUrl can read the latest values
    const valuesRef = useRef(values);
    useEffect(() => {
        valuesRef.current = values;
    }, [values]);

    const [errors, setErrors] = useState<VMRequestValidationErrors>(
        deepClone(EMPTY_VALIDATION_ERRORS),
    );
    const [allowed, setAllowed] = useState<VMRequestAllowedValues>(
        deepClone(DEFAULT_ALLOWED_VALUES),
    );

    // Fetch allowed values
    useEffect(() => {
        fetchVMOptions()
            .then((data) => {
                setAllowed((prev) => ({ ...prev, ...data }));
            })
            .catch((err) => {
                console.error("Failed to fetch VM options:", err);
            });
    }, []);

    // Persist form values to URL — called explicitly on blur / discrete interactions.
    // Uses queueMicrotask so that valuesRef is current.
    const syncToUrl = useCallback(() => {
        queueMicrotask(() => {
            const params = toSearchParams(valuesRef.current, initialValues);
            const qs = params.toString();
            const url = qs ? `?${qs}` : window.location.pathname;
            window.history.replaceState(null, "", url);
        });
    }, [initialValues]);

    const setField = useCallback(
        <K extends keyof VMRequestFormData>(
            key: K,
            value: VMRequestFormData[K],
        ) => {
            setValues((prev) => ({ ...prev, [key]: value }));
        },
        [],
    );

    const setFields = useCallback((partial: Partial<VMRequestFormData>) => {
        setValues((prev) => ({ ...prev, ...partial }));
    }, []);

    const addSshKey = useCallback(() => {
        setValues((prev) => ({
            ...prev,
            sshPubkey: [...prev.sshPubkey, ""],
        }));
    }, []);

    const removeSshKey = useCallback((index: number) => {
        setValues((prev) => {
            const next = [...prev.sshPubkey];
            next.splice(index, 1);
            return { ...prev, sshPubkey: next.length > 0 ? next : [""] };
        });
        setErrors((prev) => {
            const next = [...prev.sshPubkey];
            if (next.length > index) next.splice(index, 1);
            return { ...prev, sshPubkey: next };
        });
    }, []);

    const updateSshKey = useCallback((index: number, value: string) => {
        setValues((prev) => {
            const next = [...prev.sshPubkey];
            next[index] = value;
            return { ...prev, sshPubkey: next };
        });
    }, []);

    const isModified = useMemo(
        () => !valuesEqual(values, initialValues),
        [values, initialValues],
    );

    const reset = useCallback(() => {
        setValues(deepClone(initialValues));
        setErrors(deepClone(EMPTY_VALIDATION_ERRORS));
        window.history.replaceState(null, "", window.location.pathname);
    }, [initialValues]);

    const setValidationErrors = useCallback(
        (errs: Partial<VMRequestValidationErrors>) => {
            setErrors((prev) => ({ ...prev, ...errs }));
        },
        [],
    );

    const clearErrors = useCallback(() => {
        setErrors(deepClone(EMPTY_VALIDATION_ERRORS));
    }, []);

    const ctx = useMemo<VMRequestFormContextValue>(
        () => ({
            values,
            errors,
            allowed,
            setField,
            setFields,
            isModified,
            syncToUrl,
            reset,
            addSshKey,
            removeSshKey,
            updateSshKey,
            setValidationErrors,
            clearErrors,
        }),
        [
            values,
            errors,
            allowed,
            setField,
            setFields,
            isModified,
            syncToUrl,
            reset,
            addSshKey,
            removeSshKey,
            updateSshKey,
            setValidationErrors,
            clearErrors,
        ],
    );

    return (
        <VMRequestFormContext.Provider value={ctx}>
            {children}
        </VMRequestFormContext.Provider>
    );
}

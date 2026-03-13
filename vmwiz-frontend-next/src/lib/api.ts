import {
    VMRequestAllowedValues,
    VMRequestValidationErrors,
} from "@/lib/types/api";
import { HTTP_METHOD } from "next/dist/server/web/http";

/**
 * Callback invoked when a request needs user confirmation.
 * Receives the token the user must type, returns a promise that should resolve with the token they entered.
 */
export type OnConfirmCallback = (confirmationToken: string) => Promise<string>;

const BASE_URL = process.env.NEXT_PUBLIC_VMWIZ_BASE_URL ?? "";

/**
 * Generic backend HTTP request error
 */
export class FetchError extends Error {
    public response: Response;

    constructor(message: string, response: Response) {
        super(message);
        this.name = "FetchError";
        this.response = response;
    }
}

/**
 * Error for VM Requestvalidation errors
 */
export class ValidationError extends FetchError {
    public errors: Partial<VMRequestValidationErrors>;

    constructor(
        errors: Partial<VMRequestValidationErrors>,
        response: Response,
    ) {
        super("Validation failed", response);
        this.name = "ValidationError";
        this.errors = errors;
    }
}

interface FetchBackendOptions {
    /** HTTP method  */
    method: HTTP_METHOD;
    /** HTTP headers */
    headers?: Record<string, string>;
    /** Request body as string */
    body?: string;
    /**
     * When provided, enables confirmation-token handling.
     *
     * If the backend requires user confirmation, this callback is invoked with the token that the user is supposed to type, so that the UI can show a dialog/prompt.
     * If the first request succeeds (no confirmation needed), this callback is simply ignored.
     */
    onConfirmRequired?: OnConfirmCallback;
}

/**
 * Performs a call to the VMWiz backend.
 *
 * The output is typed as generic T.
 * Handles authentication on it's own by redirecting to SSO if a 401 is encountered.
 * Handles the token confirmation flow if `onConfirmRequired` is provided in options.
 *
 * @param path    API endpoint path, e.g. "/api/vmrequest/options"
 * @param options Request options
 * @returns Parsed structured data from the response, along with the original Response object
 * @throws FetchError if the request fails unexpectedly
 */
export async function fetchBackend<T = void>(
    path: string,
    options: FetchBackendOptions,
): Promise<{
    data: T;
    original: Response;
}> {
    const { method, headers = {}, body, onConfirmRequired } = options;

    const url = `${BASE_URL}${path}`;

    let response;
    try {
        response = await fetch(url, {
            method,
            headers,
            body,
        });
    } catch (error) {
        throw new FetchError(
            "Fetch error: " + String(error),
            new Response(null, { status: 0 }),
        );
    }

    // 401 means that the user needs to authenticate, therefore we redirect to the SSO
    if (response.status === 401) {
        const json = await response.json();
        if (json.redirectUrl) {
            window.location.href = json.redirectUrl;
        }
        throw new FetchError("Unauthorized", response);
    }

    // Something is not ok :(
    if (!response.ok) {
        const text = await response.text();

        // Handle the backend asking for a confirmation token (e.g. for destructive actions)
        if (response.status === 409 && onConfirmRequired && body) {
            // Retry with ?preview=true to obtain a confirmation token
            const { data: preview } = await fetchBackend<{
                confirmationToken: string;
            }>(`${path}?preview=true`, { method: "POST", headers, body });

            // Ask the user to confirm (may throw/reject to cancel) by calling the provided callback
            const confirmedToken = await onConfirmRequired(
                preview.confirmationToken,
            );

            // Retry the original request with the confirmed token merged in
            const parsed = JSON.parse(body) as Record<string, unknown>;
            return await fetchBackend<T>(path, {
                method: "POST",
                headers,
                body: JSON.stringify({
                    ...parsed,
                    confirmationToken: confirmedToken,
                }),
            });
        }

        throw new FetchError(
            text || `Request failed with status ${response.status}`,
            response,
        );
    }

    const text = await response.text();
    const data = text ? (JSON.parse(text) as T) : (undefined as T);
    return { data, original: response };
}

/**
 * Fetches the allowed values for VM requests (e.g. min/max CPU/RAM/Disk, OS Images, etc.) from the backend.
 * @returns The allowed values for VM requests, as provided by the backend
 */
export async function fetchVMOptions(): Promise<VMRequestAllowedValues> {
    const { data } = await fetchBackend<VMRequestAllowedValues>(
        "/api/vmrequest/options",
        { method: "GET", headers: { "Content-Type": "application/json" } },
    );
    return data;
}

/**
 * Submits a VM request to the backend.
 * @param request the VM specs to submit
 * @param onConfirmRequired See the type OnConfirmCallback for details.
 */
export async function submitVMRequest(
    formData: Record<string, unknown>,
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    try {
        await fetchBackend("/api/vmrequest", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(formData),
            onConfirmRequired,
        });
    } catch (error) {
        // In case its just validation errors, wrap them in a nicer ValidationError
        if (error instanceof FetchError && error.response.status === 403) {
            const errors = JSON.parse(
                error.message,
            ) as Partial<VMRequestValidationErrors>;
            throw new ValidationError(errors, error.response);
        }
        throw error;
    }
}

/**
 * Submits the response of a usage survey.
 * @param id the id of the survey
 * @param keep whether the user wants to keep the VM or not
 * @param onConfirmRequired See the type OnConfirmCallback for details.
 */
export async function submitSurveyResponse(
    id: string,
    keep: boolean,
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    await fetchBackend("/api/usagesurvey/set", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id, keep }),
        onConfirmRequired,
    });
}

/**
 * @param vmName the name of the VM to delete
 * @param deleteDNS whether to also delete the DNS entries for the same name
 * @param onConfirmRequired See the type OnConfirmCallback for details.
 */
export async function deleteVM(
    vmName: string,
    deleteDNS: boolean,
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    await fetchBackend("/api/vm/deleteByName", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ vmName, deleteDNS }),
        onConfirmRequired,
    });
}

export function buildBackendURL(path: string): string {
    return `${BASE_URL}${path}`;
}

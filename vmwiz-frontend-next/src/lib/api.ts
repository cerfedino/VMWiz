import {
    VMRequestAllowedValues,
    VMRequestValidationErrors,
    VMRequestListResponse,
    VMRequestEditFields,
    SurveyListResponse,
    SurveyInfo,
    SurveyResponseCategory,
    SurveyHostnameListResponse,
} from "@/lib/types/api";
import { HTTP_METHOD } from "next/dist/server/web/http";
import { getReasonPhrase } from "http-status-codes";

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

/**
 * Describes a backend request
 */
export interface BackendRequest {
    path: string;
    method: HTTP_METHOD;
    headers: Record<string, string>;
    body?: string;
}

interface FetchBackendOptions {
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
 * @param request The request descriptor (path, method, headers, body)
 * @param options Optional settings
 * @returns Parsed structured data from the response, along with the original Response object
 * @throws FetchError if the request fails unexpectedly
 */
export async function fetchBackend<T = void>(
    request: BackendRequest,
    options: FetchBackendOptions = {},
): Promise<{
    data: T;
    original: Response;
}> {
    const { path, method, headers, body } = request;
    const { onConfirmRequired } = options;

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
            }>({
                path: `${path}?preview=true`,
                method: "POST",
                headers,
                body,
            });

            // Ask the user to confirm (may throw/reject to cancel) by calling the provided callback
            const confirmedToken = await onConfirmRequired(
                preview.confirmationToken,
            );

            // Retry the original request with the confirmed token merged in
            const parsed = JSON.parse(body) as Record<string, unknown>;
            return await fetchBackend<T>({
                path,
                method: "POST",
                headers,
                body: JSON.stringify({
                    ...parsed,
                    confirmationToken: confirmedToken,
                }),
            });
        }

        throw new FetchError(
            text ||
                `Request failed with status ${response.status} ${getReasonPhrase(response.status)}`,
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
        prepareGetVMOptions(),
    );
    return data;
}
export function prepareGetVMOptions(): BackendRequest {
    return {
        path: "/api/vmrequest/options",
        method: "GET",
        headers: { "Content-Type": "application/json" },
    };
}

/**
 * Submits a VM request to the backend.
 * @param formData the VM specs to submit
 * @param onConfirmRequired See the type OnConfirmCallback for details.
 */
export async function submitVMRequest(
    formData: Record<string, unknown>,
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    try {
        await fetchBackend(prepareSubmitVMRequest(formData), {
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
export function prepareSubmitVMRequest(
    formData: Record<string, unknown>,
): BackendRequest {
    return {
        path: "/api/vmrequest",
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
    };
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
    await fetchBackend(prepareSubmitSurveyResponse(id, keep), {
        onConfirmRequired,
    });
}
export function prepareSubmitSurveyResponse(
    id: string,
    keep: boolean,
): BackendRequest {
    return {
        path: "/api/usagesurvey/set",
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id, keep }),
    };
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
    await fetchBackend(prepareDeleteVM(vmName, deleteDNS), {
        onConfirmRequired,
    });
}
export function prepareDeleteVM(
    vmName: string,
    deleteDNS: boolean,
): BackendRequest {
    return {
        path: "/api/vm/deleteByName",
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ vmName, deleteDNS }),
    };
}

export async function fetchSurveyIds(): Promise<SurveyListResponse> {
    const { data } = await fetchBackend<SurveyListResponse>(
        prepareFetchSurveyIds(),
    );
    return data;
}
export function prepareFetchSurveyIds(): BackendRequest {
    return {
        path: "/api/usagesurvey/",
        method: "GET",
        headers: { "Content-Type": "application/json" },
    };
}

export async function fetchSurveyInfo(surveyId: number): Promise<SurveyInfo> {
    const { data } = await fetchBackend<SurveyInfo>(
        prepareFetchSurveyInfo(surveyId),
    );
    return data;
}
export function prepareFetchSurveyInfo(surveyId: number): BackendRequest {
    return {
        path: `/api/usagesurvey/info?surveyId=${surveyId}`,
        method: "GET",
        headers: { "Content-Type": "application/json" },
    };
}

export async function createSurvey(
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    await fetchBackend(prepareCreateSurvey(), { onConfirmRequired });
}
export function prepareCreateSurvey(): BackendRequest {
    return {
        path: "/api/usagesurvey/create",
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
    };
}

export async function fetchSurveyResponses(
    surveyId: number,
    category: SurveyResponseCategory,
): Promise<SurveyHostnameListResponse> {
    const { data } = await fetchBackend<SurveyHostnameListResponse>(
        prepareFetchSurveyResponses(surveyId, category),
    );
    return data;
}
export function prepareFetchSurveyResponses(
    surveyId: number,
    category: SurveyResponseCategory,
): BackendRequest {
    return {
        path: `/api/usagesurvey/responses/${category}?id=${surveyId}`,
        method: "GET",
        headers: { "Content-Type": "application/json" },
    };
}

export async function resendUnsent(
    surveyId: number,
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    await fetchBackend(prepareResendUnsent(surveyId), { onConfirmRequired });
}
export function prepareResendUnsent(surveyId: number): BackendRequest {
    return {
        path: "/api/usagesurvey/resend/unsent",
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: surveyId }),
    };
}

export async function resendUnanswered(
    surveyId: number,
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    await fetchBackend(prepareResendUnanswered(surveyId), {
        onConfirmRequired,
    });
}
export function prepareResendUnanswered(surveyId: number): BackendRequest {
    return {
        path: "/api/usagesurvey/resend/unanswered",
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: surveyId }),
    };
}

/**
 * Fetches all VM requests from the backend.
 * @returns The list of VM requests
 */
export async function fetchVMRequests(): Promise<VMRequestListResponse> {
    const { data } = await fetchBackend<VMRequestListResponse>(
        prepareFetchVMRequests(),
    );
    return data;
}
export function prepareFetchVMRequests(): BackendRequest {
    return {
        path: "/api/vmrequest",
        method: "GET",
        headers: { "Content-Type": "application/json" },
    };
}

/**
 * Accepts a VM request.
 * @param id the id of the VM request to accept
 * @param onConfirmRequired See the type OnConfirmCallback for details.
 */
export async function acceptVMRequest(
    id: number,
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    await fetchBackend(prepareAcceptVMRequest(id), { onConfirmRequired });
}
export function prepareAcceptVMRequest(id: number): BackendRequest {
    return {
        path: "/api/vmrequest/accept",
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
    };
}

/**
 * Rejects a VM request.
 * @param id the id of the VM request to reject
 * @param onConfirmRequired See the type OnConfirmCallback for details.
 */
export async function rejectVMRequest(
    id: number,
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    await fetchBackend(prepareRejectVMRequest(id), { onConfirmRequired });
}
export function prepareRejectVMRequest(id: number): BackendRequest {
    return {
        path: "/api/vmrequest/reject",
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
    };
}

/**
 * Edits a VM request.
 * @param id the id of the VM request to edit
 * @param options optional fields to update (hostname, cores_cpu, ram_gb, storage_gb)
 * @param onConfirmRequired See the type OnConfirmCallback for details.
 */
export async function editVMRequest(
    id: number,
    options?: VMRequestEditFields,
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    await fetchBackend(prepareEditVMRequest(id, options), {
        onConfirmRequired,
    });
}
export function prepareEditVMRequest(
    id: number,
    options?: VMRequestEditFields,
): BackendRequest {
    const body: Record<string, unknown> = { id };
    if (options?.Hostname !== undefined) body.hostname = options.Hostname;
    if (options?.Cores !== undefined) body.cores_cpu = options.Cores;
    if (options?.RamGB !== undefined) body.ram_gb = options.RamGB;
    if (options?.DiskGB !== undefined) body.storage_gb = options.DiskGB;
    return {
        path: "/api/vmrequest/edit",
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
    };
}

export function buildBackendURL(path: string): string {
    return `${BASE_URL}${path}`;
}

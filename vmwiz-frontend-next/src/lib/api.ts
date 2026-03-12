const BASE_URL = process.env.NEXT_PUBLIC_VMWIZ_BASE_URL ?? "";

export async function fetchBackend(
    path: string,
    method: string = "GET",
    headers: Record<string, string> = {},
    body?: string,
): Promise<Response> {
    const url = `${BASE_URL}${path}`;

    const response = await fetch(url, {
        method,
        headers,
        body,
    });

    if (response.status === 401) {
        const data = await response.json();
        if (data.redirectUrl) {
            window.location.href = data.redirectUrl;
        }
        throw new Error("Unauthorized");
    }

    return response;
}

export async function fetchJSON<T>(
    path: string,
    method: string = "GET",
    headers: Record<string, string> = {},
    body?: string,
): Promise<T> {
    const response = await fetchBackend(path, method, headers, body);

    if (!response.ok) {
        const text = await response.text();
        throw new Error(
            text || `Request failed with status ${response.status}`,
        );
    }

    return response.json() as Promise<T>;
}

export async function fetchVMOptions() {
    return fetchBackend("/api/vmrequest/options", "GET", {
        "Content-Type": "application/json",
    });
}

export async function submitVMRequest(formData: Record<string, unknown>) {
    return fetchBackend(
        "/api/vmrequest",
        "POST",
        { "Content-Type": "application/json" },
        JSON.stringify(formData),
    );
}

export async function submitSurveyResponse(
    id: string,
    keep: boolean,
): Promise<Response> {
    return fetchBackend(
        "/api/usagesurvey/set",
        "POST",
        { "Content-Type": "application/json" },
        JSON.stringify({ id, keep }),
    );
}

export function buildBackendURL(path: string): string {
    return `${BASE_URL}${path}`;
}

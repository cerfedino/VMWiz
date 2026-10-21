import {
    fetchBackend,
    type BackendRequest,
    type OnConfirmCallback,
} from "@/lib/api";
import type { ResourceChangeListResponse } from "@/lib/types/resource-changes";

/**
 * Lists all VM resource change requests submitted by the SSH portal.
 */
export async function fetchResourceChanges(): Promise<ResourceChangeListResponse> {
    const { data } = await fetchBackend<ResourceChangeListResponse>({
        path: "/api/resourcechange",
        method: "GET",
        headers: { "Content-Type": "application/json" },
    });
    return data;
}

/**
 * Accepts a pending resource change request, applying it to the VM.
 * @param onConfirmRequired See OnConfirmCallback for details.
 */
export async function acceptResourceChange(
    id: number,
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    await fetchBackend(prepareAcceptResourceChange(id), { onConfirmRequired });
}
export function prepareAcceptResourceChange(id: number): BackendRequest {
    return {
        path: "/api/resourcechange/accept",
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
    };
}

/**
 * Rejects a pending resource change request without applying it.
 * @param onConfirmRequired See OnConfirmCallback for details.
 */
export async function rejectResourceChange(
    id: number,
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    await fetchBackend(prepareRejectResourceChange(id), { onConfirmRequired });
}
export function prepareRejectResourceChange(id: number): BackendRequest {
    return {
        path: "/api/resourcechange/reject",
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
    };
}

export interface ResourceChangeEditFields {
    newCores: number;
    newRamGB: number;
    newDiskGB: number;
    newSecondaryDiskGB: number;
}

/**
 * Edits a pending resource change request's target values.
 * @param onConfirmRequired See OnConfirmCallback for details.
 */
export async function editResourceChange(
    id: number,
    fields: ResourceChangeEditFields,
    onConfirmRequired?: OnConfirmCallback,
): Promise<void> {
    await fetchBackend(prepareEditResourceChange(id, fields), {
        onConfirmRequired,
    });
}
export function prepareEditResourceChange(
    id: number,
    fields: ResourceChangeEditFields,
): BackendRequest {
    return {
        path: "/api/resourcechange/edit",
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id, ...fields }),
    };
}

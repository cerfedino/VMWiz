/** GET /api/resourcechange */

export type ResourceChangeStatus =
    | "pending"
    | "auto_approved"
    | "accepted"
    | "rejected"
    | "failed"
    | "cancelled";

export interface ResourceChangeRequest {
    ID: number;
    CreatedAt: string;
    Status: ResourceChangeStatus;
    VMHostname: string;
    RequesterFingerprint: string;
    CurrentCores: number;
    CurrentRamGB: number;
    CurrentDiskGB: number;
    CurrentSecondaryDiskGB: number;
    NewCores: number;
    NewRamGB: number;
    NewDiskGB: number;
    NewSecondaryDiskGB: number;
    NeedsReviewReasons: string[];
    ProcessedAt: string | null;
    ProcessedNote: string;
    RequestReason: string;
}

export type ResourceChangeListResponse = ResourceChangeRequest[];

export interface ResourceChangeAcceptBody {
    id: number;
}
export interface ResourceChangeRejectBody {
    id: number;
}

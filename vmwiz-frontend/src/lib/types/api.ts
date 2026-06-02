export interface ConfirmationPreviewResponse {
    confirmationToken: string;
}

export interface UnauthorizedResponse {
    redirectUrl: string;
}

/** A top-level log scope (an ongoing or completed operation). */
export interface LogScope {
    id: string;
    label: string;
    startedAt: string;
    ended: boolean;
    failed: boolean;
    /** Whether the log file can be accessed or not (e.g has been deleted) */
    available: boolean;
}

/** VM Request /api/vmrequest/* */

export interface VMRequestFormData {
    email: string;
    personalEmail: string;
    isOrganization: boolean;
    orgName: string;
    hostname: string;
    image: string;
    cores: number;
    ramGB: number;
    diskGB: number;
    secondaryDiskGB: number;
    sshPubkey: string[];
    comments: string;
    accept_terms: boolean;
}

export interface VMRequestValidationErrors {
    email: string;
    personalEmail: string;
    orgName: string;
    hostname: string;
    image: string;
    cores: string;
    ramGB: string;
    diskGB: string;
    secondaryDiskGB: string;
    explanation: string;
    sshPubkey: string[];
    accept_terms: string;
}

export interface MinMax {
    min: number;
    max: number;
}

export interface VMRequestAllowedValues {
    image: string[];
    cores: MinMax;
    ramGB: MinMax;
    diskGB: MinMax;
    secondaryDiskGB: MinMax;
}

/** GET /api/vmrequest */

export type VMRequestStatus = "pending" | "accepted" | "rejected" | "hold";

export interface VMRequest {
    ID: number;
    RequestCreatedAt: string;
    RequestStatus: VMRequestStatus;
    Email: string;
    PersonalEmail: string;
    IsOrganization: boolean;
    OrgName: string;
    Hostname: string;
    Image: string;
    Cores: number;
    RamGB: number;
    DiskGB: number;
    SecondaryDiskGB: number;
    SshPubkeys: string[];
    Comments: string;
}

export type VMRequestListResponse = VMRequest[];

/** POST /api/vmrequest/accept (confirmable) */
export interface VMRequestAcceptBody {
    id: number;
    confirmationToken?: string;
}

/** POST /api/vmrequest/reject */
export interface VMRequestRejectBody {
    id: number;
}

/** POST /api/vmrequest/hold */
export interface VMRequestHoldBody {
    id: number;
}

/** POST /api/vmrequest/unhold */
export interface VMRequestUnholdBody {
    id: number;
}

/** POST /api/vmrequest/edit */
export interface VMRequestEditFields {
    Hostname?: string;
    Cores?: number;
    RamGB?: number;
    DiskGB?: number;
    SecondaryDiskGB?: number;
}

export interface VMRequestEditBody {
    id: number;
    hostname?: string;
    cores_cpu?: number;
    ram_gb?: number;
    storage_gb?: number;
}

/** POST /api/vm/deleteByName (confirmable) */
export interface VMDeleteByNameBody {
    vmName: string;
    deleteDNS: boolean;
    confirmationToken?: string;
}

/** POST /api/dns/deleteByHostname */
export interface DNSDeleteByHostnameBody {
    hostname: string;
}

/** GET /api/usagesurvey/ */
export interface SurveyListResponse {
    surveyIds: number[];
}

/** GET /api/usagesurvey/info?surveyId=<id> */
export interface SurveyInfo {
    surveyId: number;
    date: string;
    positive: number;
    negative: number;
    not_responded: number;
    not_sent: number;
}

/** POST /api/usagesurvey/create */
export interface SurveyCreateResponse {
    surveyId: number;
}

/** POST /api/usagesurvey/set */
export interface SurveySetBody {
    id: string;
    keep: boolean;
}

/** GET /api/usagesurvey/responses/{positive,negative,notsent,none}?id=<surveyId> */
export type SurveyResponseCategory =
    | "positive"
    | "negative"
    | "none"
    | "notsent";
export type SurveyHostnameListResponse = string[];

/** POST /api/usagesurvey/resend/unsent */
export interface SurveyResendUnsentBody {
    id: number;
    confirmationToken?: string;
}

/** POST /api/usagesurvey/resend/unanswered */
export interface SurveyResendUnansweredBody {
    id: number;
    confirmationToken?: string;
}

/** Default */

export const DEFAULT_FORM_VALUES: VMRequestFormData = {
    email: "",
    personalEmail: "",
    isOrganization: false,
    orgName: "",
    hostname: "",
    image: "Ubuntu 24.04 - Noble Numbat",
    cores: 2,
    ramGB: 2,
    diskGB: 15,
    secondaryDiskGB: 0,
    sshPubkey: [""],
    comments: "",
    accept_terms: false,
};

export const DEFAULT_ALLOWED_VALUES: VMRequestAllowedValues = {
    image: ["Ubuntu 24.04 - Noble Numbat", "Debian 13 - Trixie"],
    cores: { min: 1, max: 8 },
    ramGB: { min: 2, max: 16 },
    diskGB: { min: 15, max: 100 },
    secondaryDiskGB: { min: 0, max: 500 },
};

export const EMPTY_VALIDATION_ERRORS: VMRequestValidationErrors = {
    email: "",
    personalEmail: "",
    orgName: "",
    hostname: "",
    image: "",
    cores: "",
    ramGB: "",
    diskGB: "",
    secondaryDiskGB: "",
    explanation: "",
    sshPubkey: [],
    accept_terms: "",
};

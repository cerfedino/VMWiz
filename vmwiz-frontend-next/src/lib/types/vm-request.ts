// The form data submitted to POST /api/vmrequest
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

    sshPubkey: string[];

    comments: string;
    accept_terms: boolean;
}

// Validation errors returned by the backend POST /api/vmrequest
export interface VMRequestValidationErrors {
    email: string;
    personalEmail: string;
    orgName: string;
    hostname: string;
    image: string;
    cores: string;
    ramGB: string;
    diskGB: string;
    explanation: string;
    sshPubkey: string[];
    accept_terms: string;
}

// Min/max range for numeric form fields
export interface MinMax {
    min: number;
    max: number;
}

// Allowed values fetched from GET /api/vmrequest/options
export interface VMRequestAllowedValues {
    image: string[];
    cores: MinMax;
    ramGB: MinMax;
    diskGB: MinMax;
}

// Default initial form values
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

    sshPubkey: [""],

    comments: "",
    accept_terms: false,
};

// Default allowed values (overridden by backend response)
export const DEFAULT_ALLOWED_VALUES: VMRequestAllowedValues = {
    image: ["Ubuntu 24.04 - Noble Numbat", "Debian 13 - Trixie"],
    cores: { min: 1, max: 8 },
    ramGB: { min: 2, max: 16 },
    diskGB: { min: 15, max: 100 },
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
    explanation: "",
    sshPubkey: [],
    accept_terms: "",
};

// reports.go provides data structures for report generation requests

package models

// ReportSpec represents the report configuration sent from the frontend.
// Used in ReportDialog.vue when user initiates a report generation.
type ReportSpec struct {
    UserReportName        string                 `json:"user_report_name"`
    ReportType            string                 `json:"report_type"`
    DeviceIDList          []string               `json:"device_id_list"`
    DateTimeFrom          string                 `json:"datetime_from"`
    DateTimeTo            string                 `json:"datetime_to"`
    ReportOutputFieldList []string               `json:"report_output_field_list"`
    ReportOptions         map[string]interface{} `json:"report_options"`
}

// ReportRequest represents the formatted request sent to OneStepGPS API.
// Created in GenerateReportHandler by combining ReportSpec with additional options.
type ReportRequest struct {
    DateTimeFrom           string                 `json:"datetime_from"`
    DateTimeTo             string                 `json:"datetime_to"`
    DeviceIDList           []string               `json:"device_id_list"`
    ReportType             string                 `json:"report_type"`
    UserReportName         string                 `json:"user_report_name"`
    ReportOutputFieldList  []string               `json:"report_output_field_list"`
    ReportOptions          map[string]interface{} `json:"report_options"`
    ReportOptionsGeneralInfo map[string]interface{} `json:"report_options_general_info,omitempty"`
}

// ReportResponse represents OneStepGPS API's initial response to report generation.
// Used to get the report ID for status polling in GenerateReportHandler.
type ReportResponse struct {
    ReportGeneratedID string                `json:"report_generated_id"`    // ID to track report progress
    Status           string                 `json:"status"`                 // Initial status ("pending", etc.)
    Error            string                 `json:"error,omitempty"`        // Any immediate errors
    Progress         map[string]interface{} `json:"progress,omitempty"`     // Generation progress details
}

// ReportStatus represents OneStepGPS API's response to status check requests.
// Used during polling to determine when report is ready for download.
type ReportStatus struct {
    Status     string                 `json:"status"`                   // Current status ("done", "processing", etc.)
    Error      string                 `json:"error,omitempty"`          // Any errors during generation
    Progress   map[string]interface{} `json:"progress,omitempty"`       // Detailed progress information
    OutputPath string                 `json:"OutputFilePath,omitempty"` // Path to completed report
}
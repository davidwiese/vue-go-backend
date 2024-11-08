package models

// ReportSpec represents the report specification in the incoming request
type ReportSpec struct {
    UserReportName        string                 `json:"user_report_name"`
    ReportType            string                 `json:"report_type"`
    DeviceIDList          []string               `json:"device_id_list"`
    DateTimeFrom          string                 `json:"datetime_from"`
    DateTimeTo            string                 `json:"datetime_to"`
    ReportOutputFieldList []string               `json:"report_output_field_list"`
    ReportOptions         map[string]interface{} `json:"report_options"`
}

// ReportRequest represents the request structure for generating a report
type ReportRequest struct {
    DateTimeFrom           string                 `json:"datetime_from"`
    DateTimeTo             string                 `json:"datetime_to"`
    DeviceIDList          []string               `json:"device_id_list"`
    ReportType            string                 `json:"report_type"`
    UserReportName        string                 `json:"user_report_name"`
    ReportOutputFieldList []string               `json:"report_output_field_list"`
    ReportOptions         map[string]interface{} `json:"report_options"`
    ReportOptionsGeneralInfo map[string]interface{} `json:"report_options_general_info,omitempty"`
}

// ReportResponse represents the response from generating a report
type ReportResponse struct {
    ReportGeneratedID string                 `json:"report_generated_id"`
    Status           string                 `json:"status"`
    Error            string                 `json:"error,omitempty"`
    Progress         map[string]interface{} `json:"progress,omitempty"`
}

// ReportStatus represents the status response for a report
type ReportStatus struct {
    Status     string                 `json:"status"`
    Error      string                 `json:"error,omitempty"`
    Progress   map[string]interface{} `json:"progress,omitempty"`
    OutputPath string                 `json:"OutputFilePath,omitempty"`
}
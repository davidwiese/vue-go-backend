package models

// ReportSpec represents the specification for generating a report
type ReportSpec struct {
    UserReportName     string                 `json:"user_report_name"`
    ReportType         string                 `json:"report_type"`
    DeviceIDList       []string               `json:"device_id_list"`
    DateTimeFrom       string                 `json:"datetime_from"`
    DateTimeTo         string                 `json:"datetime_to"`
    ReportOutputFields []string               `json:"report_output_field_list"`
    ReportOptions      map[string]interface{} `json:"report_options"`
}

// ReportRequest represents the incoming request to generate a report
type ReportRequest struct {
    ReportSpec ReportSpec `json:"report_spec"`
}

// ReportResponse represents the response from the OneStepGPS API
type ReportResponse struct {
    ReportGeneratedID string                 `json:"report_generated_id"`
    Status           string                 `json:"status"`
    Error            string                 `json:"error"`
    Progress         map[string]interface{} `json:"progress"`
    ReportScheduledID string                 `json:"report_scheduled_id"`
    CustomerID       string                 `json:"customer_id"`
}

// ReportStatus represents the status of a report
type ReportStatus struct {
    Status     string                 `json:"status"`
    Progress   map[string]interface{} `json:"progress"`
    Error      string                 `json:"error"`
    OutputPath string                 `json:"output_file_path,omitempty"`
}
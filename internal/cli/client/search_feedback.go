package client

// SearchFeedbackRequest represents a feedback request for a prior search.
type SearchFeedbackRequest struct {
	SearchID   string `json:"search_id"`
	SelectedID string `json:"selected_id"`
	SourceType string `json:"source_type"`
}

func sendSearchFeedback(api *APIClient, searchID, selectedID, sourceType string) error {
	if api == nil || searchID == "" || selectedID == "" {
		return nil
	}
	req := SearchFeedbackRequest{
		SearchID:   searchID,
		SelectedID: selectedID,
		SourceType: sourceType,
	}
	_, err := api.Post("/search/feedback", req)
	return err
}

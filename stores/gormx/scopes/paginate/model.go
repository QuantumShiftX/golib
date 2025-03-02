package paginate

// Pagination 分页
type Pagination struct {
	Page      int64 `json:"page"`
	PageSize  int64 `json:"page_size"`
	Total     int64 `json:"total"`
	TotalPage int64 `json:"total_page"`
	Rows      any   `json:"rows"`
	Extend    any   `json:"extend,omitempty"`
}

func (p *Pagination) Offset() int64 {
	if p.Page <= 0 {
		p.Page = 1
	}
	return (p.Page - 1) * p.PageSize
}

func (p *Pagination) Limit() int64 {
	if p.PageSize < 10 {
		p.PageSize = 10
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return p.PageSize
}

func (p *Pagination) GetPage() int64 {
	return p.Page
}

func (p *Pagination) GetPageSize() int64 {
	return p.PageSize
}

func (p *Pagination) GetTotalPage() int64 {
	return p.TotalPage
}

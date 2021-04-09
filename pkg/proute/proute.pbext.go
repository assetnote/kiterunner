package proute

func (p *ProtoCrumb_UuidCrumb) UnwrapCrumb() Crumb         { return *p.UuidCrumb }
func (p *ProtoCrumb_StaticCrumb) UnwrapCrumb() Crumb       { return *p.StaticCrumb }
func (p *ProtoCrumb_IntCrumb) UnwrapCrumb() Crumb          { return *p.IntCrumb }
func (p *ProtoCrumb_BoolCrumb) UnwrapCrumb() Crumb         { return *p.BoolCrumb }
func (p *ProtoCrumb_FloatCrumb) UnwrapCrumb() Crumb        { return *p.FloatCrumb }
func (p *ProtoCrumb_RandomStringCrumb) UnwrapCrumb() Crumb { return *p.RandomStringCrumb }
func (p *ProtoCrumb_RegexStringCrumb) UnwrapCrumb() Crumb  { return *p.RegexStringCrumb }
func (p *ProtoCrumb_BasicAuthCrumb) UnwrapCrumb() Crumb    { return *p.BasicAuthCrumb }
func (p *ProtoCrumb_ArrayCrumb) UnwrapCrumb() Crumb        { return *p.ProuteCrumb()}
func (p *ProtoCrumb_ObjectCrumb) UnwrapCrumb() Crumb       { return *p.ProuteCrumb() }
func (p *ProtoCrumb_StringCrumbCrumb) UnwrapCrumb() Crumb  { return *p.ProuteCrumb() }

func (p *ProtoCrumb_ArrayCrumb) ProuteCrumb() *ArrayCrumb {
	if p == nil {
		return nil
	}
	return &ArrayCrumb{
		Name: p.ArrayCrumb.Name,
		Element: p.ArrayCrumb.Element.GetRawCrumb(),
	}
}

func (p *ProtoCrumb_ObjectCrumb) ProuteCrumb() *ObjectCrumb {
	if p == nil {
		return nil
	}
	return &ObjectCrumb{
		Name: p.ObjectCrumb.Name,
		Elements: FromProtoCrumbs(p.ObjectCrumb.Elements),
	}
}

func (p *ProtoCrumb_StringCrumbCrumb) ProuteCrumb() *StringCrumbCrumb {
	if p == nil {
		return nil
	}
	return &StringCrumbCrumb{
		Name: p.StringCrumbCrumb.Name,
		Child: p.StringCrumbCrumb.Child.GetRawCrumb(),
	}
}

type UnwrapCrumber interface {
	UnwrapCrumb() Crumb
}

func FromCrumbs(in []Crumb) (ret []ProtoCrumb) {
	for _, v := range in {
		ret = append(ret, *v.protoCrumb())
	}
	return ret
}

func (m *ProtoCrumb) GetRawCrumb() Crumb {
	if m != nil {
		tmp := m.Crumb.(UnwrapCrumber).UnwrapCrumb()
		return tmp
	}
	return nil
}

func FromProtoCrumbs(in []ProtoCrumb) (ret []Crumb) {
	for _, v := range in {
		ret = append(ret, v.GetRawCrumb())
	}
	return ret
}

func (p ProtoRoute) Route() Route {
	return Route{
		TemplatePath: p.TemplatePath,
		Method:       p.Method,
		ContentType:  p.ContentType,
		PathCrumbs:   FromProtoCrumbs(p.PathCrumbs),
		HeaderCrumbs: FromProtoCrumbs(p.HeaderCrumbs),
		QueryCrumbs:  FromProtoCrumbs(p.QueryCrumbs),
		BodyCrumbs:   FromProtoCrumbs(p.BodyCrumbs),
	}
}

func (p Route) ProtoRoute() ProtoRoute {
	return ProtoRoute{
		TemplatePath: p.TemplatePath,
		Method:       p.Method,
		ContentType:  p.ContentType,
		PathCrumbs:   FromCrumbs(p.PathCrumbs),
		HeaderCrumbs: FromCrumbs(p.HeaderCrumbs),
		QueryCrumbs:  FromCrumbs(p.QueryCrumbs),
		BodyCrumbs:   FromCrumbs(p.BodyCrumbs),
	}
}

func FromProtoRoutes(in []ProtoRoute) (ret []Route) {
	for _, v := range in {
		ret = append(ret, v.Route())
	}
	return ret
}

func FromRoutes(in []Route) (ret []ProtoRoute) {
	for _, v := range in {
		ret = append(ret, v.ProtoRoute())
	}
	return ret
}

func (p ProtoAPI) API() API {
	return API{
		URL:          p.URL,
		ID:           p.ID,
		Routes:       FromProtoRoutes(p.Routes),
		CookieCrumbs: FromProtoCrumbs(p.CookieCrumbs),
		HeaderCrumbs: FromProtoCrumbs(p.HeaderCrumbs),
		QueryCrumbs:  FromProtoCrumbs(p.QueryCrumbs),
		BodyCrumbs:   FromProtoCrumbs(p.BodyCrumbs),
	}
}

func (p API) ProtoAPI() ProtoAPI {
	return ProtoAPI{
		URL:          p.URL,
		ID:           p.ID,
		Routes:       FromRoutes(p.Routes),
		CookieCrumbs: FromCrumbs(p.CookieCrumbs),
		HeaderCrumbs: FromCrumbs(p.HeaderCrumbs),
		QueryCrumbs:  FromCrumbs(p.QueryCrumbs),
		BodyCrumbs:   FromCrumbs(p.BodyCrumbs),
	}
}

func (p ProtoAPIS) APIS() APIS {
	ret := APIS{}
	for _, v := range p.APIs {
		ret = append(ret, v.API())
	}
	return ret
}

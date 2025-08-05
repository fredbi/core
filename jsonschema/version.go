package jsonschema

// Version describes a recognized dialect of JSON schema.
type Version uint8

const (
	VersionUndefined Version = iota
	VersionDraft4
	VersionDraft5 // same a VersionDraft4 (only spec clarifications)
	VersionDraft6
	VersionDraft7
	VersionDraft2019
	VersionDraft2020
	VersionOpenAPIv2       // extends VersionDraft4
	VersionOpenAPIv2Simple // constrained for Parameter and Header
	VersionOpenAPIv300     // extends VersionDraft5
	VersionOpenAPIv301     // TODO = v300
	VersionOpenAPIv302
	VersionOpenAPIv303
	VersionOpenAPIv304
	VersionOpenAPIv310 // extends VersionDraft2020
	VersionOpenAPIv311
	VersionOpenAPIv4Draft
)

func (v Version) String() string {
	switch v {
	case VersionUndefined:
		return "undefined"
	case VersionDraft4:
		return "draft4"
	case VersionDraft5:
		return "draft5"
	case VersionDraft6:
		return "draft6"
	case VersionDraft7:
		return "draft7"
	case VersionDraft2019:
		return "draft2019"
	case VersionDraft2020:
		return "draft2020"
	case VersionOpenAPIv2:
		return "openapi v2"
	case VersionOpenAPIv2Simple:
		return "openapi v2 simple"
	case VersionOpenAPIv300:
		return "openapi v3.0.0"
	case VersionOpenAPIv301:
		return "openapi v3.0.1"
	case VersionOpenAPIv302:
		return "openapi v3.0.2"
	case VersionOpenAPIv303:
		return "openapi v3.0.3"
	case VersionOpenAPIv304:
		return "openapi v3.0.4"
	case VersionOpenAPIv310:
		return "openapi v3.1.0"
	case VersionOpenAPIv311:
		return "openapi v3.1.1"
	case VersionOpenAPIv4Draft:
		return "openapi v4 draft"
	default:
		panic("unsupported jsonschema dialect version")
	}
}

// MetaSchemaURL gives the URL used to identify a schema version.
//
// For JSON schema drafts, see https://json-schema.org/draft/2019-09/schema.
//
// For OpenAPI schemas, see https://spec.openapis.org/oas.
//
// Notice that OpenAPI schema do not necesarily resolve with a meta-schema like all json-schema.org drafts.
func (v Version) MetaSchemaURL() string {
	switch v {
	case VersionUndefined:
		return ""
	case VersionDraft4:
		return "https://json-schema.org/draft-04/schema#"
	case VersionDraft5:
		return "https://json-schema.org/draft-04/schema#"
	case VersionDraft6:
		return "https://json-schema.org/draft-06/schema#"
	case VersionDraft7:
		return "https://json-schema.org/draft-07/schema#"
	case VersionDraft2019:
		return "https://json-schema.org/draft/2019-09/schema"
	case VersionDraft2020:
		return "https://json-schema.org/draft/2020-12/schema"
	case VersionOpenAPIv2:
		return "https://spec.openapis.org/oas/2.0/schema"
	case VersionOpenAPIv2Simple:
		return ""
	case VersionOpenAPIv300:
		return "https://spec.openapis.org/oas/3.0/schema"
	case VersionOpenAPIv301:
		return "https://spec.openapis.org/oas/3.0/schema"
	case VersionOpenAPIv302:
		return "https://spec.openapis.org/oas/3.0/schema"
	case VersionOpenAPIv303:
		return "https://spec.openapis.org/oas/3.0/schema"
	case VersionOpenAPIv304:
		return "https://spec.openapis.org/oas/3.0/schema"
	case VersionOpenAPIv310:
		return "https://spec.openapis.org/oas/3.1/dialect/2024-11-10"
	case VersionOpenAPIv311:
		return "https://spec.openapis.org/oas/3.1/dialect/2024-11-10"
	case VersionOpenAPIv4Draft:
		return ""
	default:
		panic("unsupported jsonschema dialect version")
	}
}

// Less compare the schema version of v with the version of vv.
//
// It returns true is the version of vv is prior to the version of v.
func (v Version) Less(vv Version) bool {
	const (
		highestDraft = int(VersionDraft2020)
		lowestOAI    = int(VersionOpenAPIv2)
	)

	switch {
	case int(v) <= highestDraft && int(vv) <= highestDraft:
		return int(v) < int(vv)
	case int(v) >= lowestOAI && int(vv) >= lowestOAI:
		return int(v) < int(vv)
	case int(v) >= lowestOAI && int(v) < int(VersionOpenAPIv310):
		if int(vv) <= int(VersionDraft5) {
			return false
		}
		return true
	case int(v) >= int(VersionOpenAPIv310):
		if int(vv) <= int(VersionDraft2020) {
			return false
		}
		return true
	case int(vv) >= lowestOAI && int(vv) < int(VersionOpenAPIv310):
		if int(v) <= int(VersionDraft5) {
			return true
		}
		return false
	case int(vv) >= int(VersionOpenAPIv310):
		if int(v) <= int(VersionDraft5) {
			return true
		}
		return false
	default:
		return false
	}
}

// DocURL yields the URL of the official specification for every supported schema.
func (v Version) DocURL() string {
	switch v {
	case VersionUndefined:
		return "undefined"
	case VersionDraft4:
		return "draft4" // TODO
	case VersionDraft5:
		return "draft5"
	case VersionDraft6:
		return "draft6"
	case VersionDraft7:
		return "draft7"
	case VersionDraft2019:
		return "draft2019"
	case VersionDraft2020:
		return "draft2020"
	case VersionOpenAPIv2:
		return "https://spec.openapis.org/oas/v2.0.html"
	case VersionOpenAPIv300:
		return "https://spec.openapis.org/oas/v3.0.0.html"
	case VersionOpenAPIv301:
		return "https://spec.openapis.org/oas/v3.0.1.html"
	case VersionOpenAPIv302:
		return "https://spec.openapis.org/oas/v3.0.2.html"
	case VersionOpenAPIv303:
		return "https://spec.openapis.org/oas/v3.0.3.html"
	case VersionOpenAPIv304:
		return "https://spec.openapis.org/oas/v3.0.4.html"
	case VersionOpenAPIv310:
		return "openapi v3.1.0"
	default:
		panic("unsupported jsonschema dialect version")
	}
}

// VersionRequirements specifies for a [Schema] what are the MinVersion and the MaxVersion supported.
type VersionRequirements struct {
	MinVersion          Version
	MinOAIVersion       Version
	MaxVersion          Version
	MaxOAIVersion       Version
	StrictMaxVersion    Version
	StrictMaxOAIVersion Version
}

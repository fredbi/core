package structural

// Option customizes the behavior of the [SchemaAnalyzer].
type Option func(*options)

type options struct {
	bundleOptions

	withValidations bool
	withReportAudit bool // will report audit records as a "x-go-audit" extension

	explicit              bool
	liftAnonymousAllOf    bool
	pushArrayAllOf        bool
	reduce                bool
	simplify              bool
	uniqueCompositions    bool
	pruneEnums            bool
	splitOverlappingAllOf bool
	splitTypes            bool
	refactEnums           bool
	refactIfThenElse      bool
	extensionMappers      []ExtensionMapper
}

func applyOptionsWithDefaults(opts []Option) options {
	o := options{}

	for _, apply := range opts {
		apply(&o)
	}

	return o
}

// WithAnalyzeValidations instructs the [SchemaAnalyzer] to carry out further analysis on validations.
func WithAnalyzeValidations(enabled bool) Option {
	return func(o *options) {
		o.withValidations = enabled
	}
}

// WithReportAudit tells the analyzer to include an audit record of schema transforms when
// using [SchemaAnalyzer.Dump] or [SchemaAnalyzer.MarshalJSON].
func WithReportAudit(enabled bool) Option {
	return func(o *options) {
		o.withReportAudit = enabled
	}
}

// WithExtensionMappers adds chained mappers to the analyzer, so that it may validate and convert extensions.
func WithExtensionMappers(mappers ...ExtensionMapper) Option {
	return func(o *options) {
		o.extensionMappers = append(o.extensionMappers, mappers...)
	}
}

// WithLiftAnonymousAllOf instructs the [SchemaAnalyzer] to rewrite schemas with anonymous allOf members
// by lifting the anonymous parts to their parent, whenever possible.
//
// Exception: if we have [WithRefactorEnums] enabled (which performs the exact opposite operation for enums),
// the lifting of enums is not carried out.
//
// Example:
//
// The schema:
//
//	parent:
//	  type: object
//	  properties:
//	    {props parent}
//		allOf:
//		  - $ref: '#/$defs/namedSchema
//		  - type: object
//		    properties: {props child}
//
// Is transformed into:
//
//	parent:
//	  type: object
//	  properties:
//	    {props parent}
//	    {props child}
//	  allOf:
//	    - $ref: '#/$defs/namedSchema
//
// It {props parent} and {props child} have propertie in common, these are merged in the refactored result.
func WithLiftAnonymousAllOf(enabled bool) Option {
	return func(o *options) {
		o.liftAnonymousAllOf = enabled
	}
}

// WithSplitOverlappingAllOf instructs the analyzer to rewrite "allOf" objects with overlapping properties so
// the resulting "allOf" members do not overlap.
//
// This refacting resolves overlaps with the parent's properties and overlaps between members.
//
// Overlapping named objects produce new split definitions.
//
// Overlapping properties of anonymous allOf members are lifted into the parent.
//
// The new named definition constructed are named by calling the [NameProvider] specified.
//
// Example:
//
// The schema:
//
//					parent:
//				    type: object
//			     properties:
//			       a:
//			         type: string
//		           maxLength: 5
//			       {{ props parent }}
//			     allOf:
//		        - $ref: #/$defs/part1
//		        - $ref: #/$defs/part2
//		        - type: object
//		          properties:
//		            a:
//		              type: string
//		              minLength: 1
//		            b:
//		              type: integer
//		            c:
//		              type: integer
//		     $defs:
//		        part1:
//		          type: object
//		          properties:
//		            a:
//		              type: string
//		              minLength: 1
//	                pattern:  '^[a-z]$'
//		            b:
//		              type: integer
//		              maximum: 1000
//		            d:
//		              type: string
//		        part2:
//		          type: object
//		          properties:
//		            a:
//		              type: string
//		              minLength: 3
//		            b:
//		              type: integer
//		              minimum: 100
//		            e:
//		              type: string
//
// Is transformed into:
//
//							parent:
//					      properties:
//						      a:
//					          type: string
//							      maxLength: 5
//							      minLength: 3    # <- merged validation
//				            pattern:  '^[a-z]$'
//								  {{ props parent }}
//								allOf:
//				          - type: object
//				            properties:      # <- overlapping properties with sibling allOf members
//			                b:
//							          type: integer
//					              maximum: 1000
//					              minimum: 100
//							    - $ref: #/$defs/part1WithoutAB
//							    - $ref: #/$defs/part2WithoutAB
//							    - type: object    # <- overlapping property from anonymous member lifted and merged
//							      properties:
//							        c:
//							          type: string
//						  $defs:
//							  part1:
//				          type: object
//				          properties:
//				            a:
//				              type: string
//				              minLength: 1
//			                pattern:  '^[a-z]$'
//				            b:
//				              type: integer
//				              maximum: 1000
//	                allOf:
//	                  - $ref: #/$defs/part1WithoutAB
//							  part2:
//				            a:
//				              type: string
//				              minLength: 3
//				            b:
//				              type: integer
//				              minimum: 100
//	                allOf:
//	                  - $ref: #/$defs/part2WithoutAB
//							  part1WithoutAB:
//		  		        type: object
//				          properties:
//				            d:
//				              type: string
//							  part2WithoutAB:
//		  		        type: object
//				          properties:
//		 		          e:
//			  	            type: string
func WithSplitOverlappingAllOf(enabled bool) Option {
	return func(o *options) {
		o.splitOverlappingAllOf = enabled
	}
}

// WithPushArrayAllOf instructs the [SchemaAnalyzer] to rewrite array schemas with an allOf composition
// to push the allOf down on items and merge the array validation.
//
// Example:
// The schema:
//
//			parent:
//			  type: array
//			  maxItems: 10
//			  items:
//			    { items 0 }
//			  allOf:
//		      - type: array
//		        item: { items 1 }
//	          minItems: 5
//		      - type: array
//		        item: { items 2 }
//	          uniqueItems: true
//
// Is transformed into:
//
//			parent:
//			  type: array
//	 		  maxItems: 10
//	      minItems: 5
//	      uniqueItems: true
//				items:
//			    allOf:
//				    - { items 0 }
//		        - { items 1 }
//		        - { items 2 }
func WithPushArrayAllOf(enabled bool) Option {
	return func(o *options) {
		o.pushArrayAllOf = enabled
	}
}

// WithUniqueCompositions instructs the [SchemaAnalyzer] to rewrite schemas with multiple adjactent compositions
// like "allOf", "oneOf", "anyOf" so that each schema only gets one of those.
func WithUniqueCompositions(enabled bool) Option {
	return func(o *options) {
		o.uniqueCompositions = enabled
	}
}

// WithReduceValidations instructs the [SchemaAnalyzer] to reduce validations that may be evaluated as
// always false or always true:
//
// * always true schemas are replaced by "true"
// * always false schemas are reduced to "false"
func WithReduceValidations(enabled bool) Option {
	return func(o *options) {
		o.liftAnonymousAllOf = enabled
	}
}

// WithSimplifyValidations instructs the [SchemaAnalyzer] to simplify validation expressions.
//
// Simplifications carried out:
//   - reduce validation expressions that evaluate to true or false.
//   - remove "true" members from allOf
//   - ~remove oneOf with a "true" member~ NOT SURE
//   - ~remove anyOf with a "true" member~ NOT SURE
//   - lift compositions that reduce to one anonymous member (allOf, anyOf, oneOf)
//   - rewrite anyOf as allOf when all members are always simultaneously true
//   - rewrite anyOf as oneOf when all members can never be simultaneously true
//   - enum: null value for null type is simplified when type is null
func WithSimplifyValidations(enabled bool) Option {
	return func(o *options) {
		o.simplify = enabled
	}
}

// WithExplicitValidations instructs the [SchemaAnalyzer] to render all JSON schema defaults explicitly.
//
// This is the case in particular for "additionalProperties" in objects, "items" in tuples (aka "additionalItems"),
// missing "items" in arrays.
//
// Empty schemas are normalized as "true".
func WithExplicitValidations(enabled bool) Option {
	return func(o *options) {
		o.explicit = enabled
	}
}

// WithPruneEnums instructs the [SchemaAnalyzer] to prune from enum values all values that do not pass other validations
// for this schema.
//
// Example:
//
// The schema:
//
//	parent
//	  type: integer
//	  maximum: 100
//	  enum:
//	    - 1
//	    - 10
//	    - 1000
//
// Is transformed into:
//
//		parent:
//	    type: integer
//	    maximum: 100
//	    enum:
//	      - 1
//	      - 10
func WithPruneEnums(enabled bool) Option {
	return func(o *options) {
		o.pruneEnums = enabled
	}
}

// WithSplitMultipleTypes refactor all multiple type definition with a "oneOf" or "anyOf" replacement.
//
// Child validations are pushed according to the type they apply to.
//
// "anyOf" is used when we have compatible types, i.e. "number" and "integer".
//
// When this option is enabled, no schema has multiple types.
//
// Exceptions:
//   - schema with no constraint at all (no type, no typed validation) are not transformed
//   - enum validation are not assigned to a specific type and therefore not pushed down, unless their type may be infered
//     (see example below)
//
// Example:
//
// The schema:
//
//				parent
//				  type: [integer, string, object,null]
//			    minimum: 20
//			    maxLength: 5
//			    minProperties: 1
//		      enum:
//		        - 30
//		        - "a"
//		        - "b"
//		        - {"a": 1, "b": 2}
//		        - 40
//	          - null
//
// Is transformed into:
//
//				parent:
//			   oneOf:
//			     - type: integer
//			       minimum: 20
//		         enum:
//		           - 30
//		           - 40
//			     - type: string
//			       maxLength: 5
//		         enum:
//		           - "a"
//		           - "b"
//			     - type: object
//			       minProperties: 1
//		         enum:
//			        - {"a": 1, "b": 2}
//			     - type: null
//	           const: "null"
func WithSplitMultipleTypes(enabled bool) Option {
	return func(o *options) {
		o.splitTypes = enabled
	}
}

// WithRefactEnums instructs the [SchemaAnalyzer] to refactor enum declarations as a separate anonymous allOf member.
//
// This is useful to regroup enum validations a distinct types and clarify the rest of the schema.
//
// If the parent may have multiple types, this refactoring may take place if the mutiple types refactoring has
// been applied beforehand. See [WithSplitMultipleTypes].
//
// Example:
//
// The schema:
//
//	parent:
//	  { ... }
//	  enum:
//	    - value1
//	    - value2
//	    - value3
//
// Is refactored into:
//
//	parent:
//		{ ... }
//		allOf:
//		  - type: { type of parent }
//	      description: 'enumerates the valid values for a {parent}'
//		    enum:
//		 	    - value1
//			    - value2
//		 	    - value3
func WithRefactorEnums(enabled bool) Option {
	return func(o *options) {
		o.refactEnums = enabled
	}
}

// WithRefactorIfThenElse rewrites 'if', 'then', 'else' constructs into 'oneOf' and 'allOf'.
//
// Example:
//
// The schema:
//
//		parent:
//		  { ... }
//		  if: { if schema }
//		  then: { then schema }
//	   else: { else schema }
//
// Is refactored into:
//
//			parent:
//				{ ... }
//		   oneOf:
//		     - allOf:
//		         - { if schema }
//		         - { then schema }
//		     - allOf:
//		         - not { if schema }
//		         - { else schema }
//
//	 If there is no 'else' clause, this yields:
//
//			parent:
//				{ ... }
//		   oneOf:
//		     - allOf:
//		         - { if schema }
//		         - { then schema }
//		     - not { if schema }
func WithRefactorIfThenElse(enabled bool) Option {
	return func(o *options) {
		o.refactIfThenElse = enabled
	}
}

// WithRefactorSchemas instructs the [SchemaAnalyzer] to refactor schemas by applying options:
//
//   - [WithAnalyzeValidations]
//   - [WithExplicitValidations]
//   - [WithLiftAnonymousAllOf]
//   - [WithPruneEnums]
//   - [WithPushArrayAllOf]
//   - [WithReduceValidations]
//   - [WithRefactorIfThenElse]
//   - [WithSimplifyValidations]
//   - [WithSplitMultipleTypes]
//   - [WithSplitOverlappingAllOf]
//   - [WithUniqueCompositions]
func WithRefactorSchemas(enabled bool) Option {
	return func(o *options) {
		o.explicit = true
		o.liftAnonymousAllOf = enabled
		o.pruneEnums = enabled
		o.pushArrayAllOf = enabled
		o.reduce = enabled
		o.refactIfThenElse = enabled
		o.simplify = enabled
		o.splitOverlappingAllOf = enabled
		o.splitTypes = enabled
		o.uniqueCompositions = enabled
		o.withValidations = enabled
	}
}

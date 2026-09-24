package crud

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/jinzhu/inflection"
)

// Options encapsulates generator arguments.
type Options struct {
	ModelPath  string
	StructName string
}

// Generate scaffolds CRUD module artifacts based on an existing GORM model.
func Generate(opts Options) error {
	if strings.TrimSpace(opts.ModelPath) == "" {
		return errors.New("model path is required")
	}

	absPath, err := filepath.Abs(opts.ModelPath)
	if err != nil {
		return fmt.Errorf("resolve model path: %w", err)
	}

	fset := token.NewFileSet()
	fileAST, err := parser.ParseFile(fset, absPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse model file: %w", err)
	}

	structType, structName, err := findStruct(fileAST, opts.StructName)
	if err != nil {
		return err
	}

	imports := collectImports(fileAST)
	fields, err := extractFields(fset, structType, imports)
	if err != nil {
		return err
	}

	idField, hasID := findIDField(fields)
	if !hasID {
		return errors.New("model must define an ID field named 'ID'")
	}

	entity := structName
	entityLower := strcase.ToLowerCamel(entity)
	entitySnake := strcase.ToSnake(entity)
	entityPluralSnake := inflection.Plural(entitySnake)
	entityPluralVar := strcase.ToLowerCamel(entityPluralSnake)
	entityPluralTitle := strings.Title(entityPluralSnake)

	createFields, updateFields, responseFields := buildDTOFields(fields)
	createAssignments := buildCreateAssignments(fields)
	updateStatements := buildUpdateStatements(fields)
	responseMappings := buildResponseMappings(fields)

	dtoImports := buildDTOImports(fields, imports)
	repoImports := buildRepositoryImports()
	serviceImports := buildServiceImports()
	controllerImports := buildControllerImports()
	idSpec := buildIDSpec(idField, imports)
	controllerImports = append(controllerImports, idSpec.ExtraImports...)
	repoImports = append(repoImports, idSpec.ExtraImports...)
	serviceImports = append(serviceImports, idSpec.ExtraImports...)

	moduleImports := []importSpec{
		{Path: "github.com/gin-gonic/gin"},
		{Path: "go-base/internal/httpserver/middleware"},
		{Path: "go-base/internal/httpserver/modules"},
	}

	data := templateData{
		Package:           fileAST.Name.Name,
		Entity:            entity,
		EntityLower:       entityLower,
		EntitySnake:       entitySnake,
		EntityPlural:      entityPluralSnake,
		EntityPluralTitle: entityPluralTitle,
		EntityPluralVar:   entityPluralVar,
		CreateFields:      createFields,
		UpdateFields:      updateFields,
		ResponseFields:    responseFields,
		CreateAssignments: createAssignments,
		UpdateStatements:  updateStatements,
		ResponseMappings:  responseMappings,
		IDSpec:            idSpec,
		DTOImports:        uniqueImports(dtoImports),
		RepositoryImports: uniqueImports(repoImports),
		ServiceImports:    uniqueImports(serviceImports),
		ControllerImports: uniqueImports(controllerImports),
		ModuleImports:     uniqueImports(moduleImports),
	}

	outputDir := filepath.Dir(absPath)
	files := map[string]string{
		fmt.Sprintf("%s_dto_gen.go", entitySnake):        dtoTemplate,
		fmt.Sprintf("%s_repository_gen.go", entitySnake): repositoryTemplate,
		fmt.Sprintf("%s_service_gen.go", entitySnake):    serviceTemplate,
		fmt.Sprintf("%s_controller_gen.go", entitySnake): controllerTemplate,
		fmt.Sprintf("%s_module_gen.go", entitySnake):     moduleTemplate,
	}

	rendered := make(map[string][]byte, len(files))
	for filename, tmpl := range files {
		destPath := filepath.Join(outputDir, filename)
		if _, err := os.Stat(destPath); err == nil {
			return fmt.Errorf("file %s already exists", filename)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("check %s: %w", filename, err)
		}

		formatted, err := renderTemplate(tmpl, data)
		if err != nil {
			return fmt.Errorf("render %s: %w", filename, err)
		}

		rendered[filename] = formatted
	}

	for filename, formatted := range rendered {
		destPath := filepath.Join(outputDir, filename)
		if err := os.WriteFile(destPath, formatted, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", filename, err)
		}
	}

	return nil
}

type fieldInfo struct {
	Name             string
	Type             string
	BaseType         string
	JSONName         string
	ValidateTag      string
	IsPointer        bool
	IncludeInCreate  bool
	IncludeInUpdate  bool
	IncludeInResp    bool
	IsID             bool
	IsTimestampField bool
	Aliases          []string
}

type dtoField struct {
	Name string
	Type string
	Tag  string
}

type responseField struct {
	Name string
	Type string
	Tag  string
}

type idSpec struct {
	Type         string
	VarName      string
	ParseCode    string
	ExtraImports []importSpec
	ZeroValue    string
}

type templateData struct {
	Package           string
	Entity            string
	EntityLower       string
	EntitySnake       string
	EntityPlural      string
	EntityPluralTitle string
	EntityPluralVar   string
	CreateFields      []dtoField
	UpdateFields      []dtoField
	ResponseFields    []responseField
	CreateAssignments []string
	UpdateStatements  []string
	ResponseMappings  []string
	IDSpec            idSpec
	DTOImports        []importSpec
	RepositoryImports []importSpec
	ServiceImports    []importSpec
	ControllerImports []importSpec
	ModuleImports     []importSpec
}

type importSpec struct {
	Alias string
	Path  string
}

var templateFuncs = template.FuncMap{
	"backtick": func(tag string) string {
		if tag == "" {
			return ""
		}
		return "`" + tag + "`"
	},
}

func renderTemplate(tmplSrc string, data templateData) ([]byte, error) {
	tmpl, err := template.New("crud").Funcs(templateFuncs).Parse(tmplSrc)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.Bytes(), nil
	}

	return formatted, nil
}

func findStruct(file *ast.File, desired string) (*ast.StructType, string, error) {
	var structType *ast.StructType
	var structName string

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			if desired != "" {
				if typeSpec.Name.Name == desired {
					return st, typeSpec.Name.Name, nil
				}
				continue
			}

			if structType != nil {
				return nil, "", errors.New("multiple structs found, specify --struct")
			}

			structType = st
			structName = typeSpec.Name.Name
		}
	}

	if structType == nil {
		if desired != "" {
			return nil, "", fmt.Errorf("struct %s not found", desired)
		}
		return nil, "", errors.New("no struct definitions found in model file")
	}

	return structType, structName, nil
}

func collectImports(file *ast.File) map[string]string {
	imports := make(map[string]string)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			alias = defaultAliasForPath(path)
		}
		imports[alias] = path
	}
	return imports
}

func extractFields(fset *token.FileSet, structType *ast.StructType, imports map[string]string) ([]fieldInfo, error) {
	var fields []fieldInfo

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		name := field.Names[0].Name

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, field.Type); err != nil {
			return nil, fmt.Errorf("print field type: %w", err)
		}
		typeStr := buf.String()

		jsonName := strcase.ToLowerCamel(name)
		validateTag := ""
		if field.Tag != nil {
			tagValue := strings.Trim(field.Tag.Value, "`")
			tag := reflect.StructTag(tagValue)
			if tagJSON := tag.Get("json"); tagJSON != "" {
				parts := strings.Split(tagJSON, ",")
				if parts[0] == "-" {
					continue
				}
				if parts[0] != "" {
					jsonName = parts[0]
				}
			}
			validateTag = tag.Get("validate")
		}

		isPointer := strings.HasPrefix(typeStr, "*")
		baseType := strings.TrimPrefix(typeStr, "*")

		info := fieldInfo{
			Name:             name,
			Type:             typeStr,
			BaseType:         baseType,
			JSONName:         jsonName,
			ValidateTag:      validateTag,
			IsPointer:        isPointer,
			IncludeInCreate:  includeInCreate(name),
			IncludeInUpdate:  includeInUpdate(name),
			IncludeInResp:    includeInResponse(name),
			IsID:             name == "ID",
			IsTimestampField: name == "CreatedAt" || name == "UpdatedAt",
			Aliases:          collectTypeAliases(field.Type),
		}

		fields = append(fields, info)
	}

	return fields, nil
}

func includeInCreate(name string) bool {
	switch name {
	case "ID", "CreatedAt", "UpdatedAt", "DeletedAt":
		return false
	default:
		return true
	}
}

func includeInUpdate(name string) bool {
	return includeInCreate(name)
}

func includeInResponse(name string) bool {
	return name != "DeletedAt"
}

func collectTypeAliases(expr ast.Expr) []string {
	set := map[string]struct{}{}
	collectAliases(expr, set)
	aliases := make([]string, 0, len(set))
	for alias := range set {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

func collectAliases(expr ast.Expr, set map[string]struct{}) {
	switch v := expr.(type) {
	case *ast.SelectorExpr:
		if ident, ok := v.X.(*ast.Ident); ok {
			set[ident.Name] = struct{}{}
		}
	case *ast.StarExpr:
		collectAliases(v.X, set)
	case *ast.ArrayType:
		collectAliases(v.Elt, set)
	case *ast.MapType:
		collectAliases(v.Key, set)
		collectAliases(v.Value, set)
	case *ast.IndexExpr:
		collectAliases(v.X, set)
		collectAliases(v.Index, set)
	case *ast.IndexListExpr:
		collectAliases(v.X, set)
		for _, idx := range v.Indices {
			collectAliases(idx, set)
		}
	case *ast.Ident:
		// no-op
	default:
		// ignore other forms
	}
}

func findIDField(fields []fieldInfo) (fieldInfo, bool) {
	for _, f := range fields {
		if f.IsID {
			return f, true
		}
	}
	return fieldInfo{}, false
}

func buildDTOFields(fields []fieldInfo) ([]dtoField, []dtoField, []responseField) {
	var createFields []dtoField
	var updateFields []dtoField
	var responseFields []responseField

	for _, f := range fields {
		if f.IncludeInCreate {
			createFields = append(createFields, dtoField{
				Name: f.Name,
				Type: createFieldType(f),
				Tag:  buildTag(f.JSONName, f.ValidateTag, false),
			})
		}

		if f.IncludeInUpdate {
			updateFields = append(updateFields, dtoField{
				Name: f.Name,
				Type: updateFieldType(f),
				Tag:  buildTag(f.JSONName, buildUpdateValidateTag(f.ValidateTag), true),
			})
		}

		if f.IncludeInResp {
			responseFields = append(responseFields, responseField{
				Name: f.Name,
				Type: f.Type,
				Tag:  buildResponseTag(f.JSONName, f.IsPointer || f.IsTimestampField),
			})
		}
	}

	return createFields, updateFields, responseFields
}

func createFieldType(f fieldInfo) string {
	if f.IsPointer {
		return f.Type
	}
	return f.Type
}

func updateFieldType(f fieldInfo) string {
	if f.IsPointer {
		return f.Type
	}
	return "*" + f.Type
}

func buildTag(jsonName, validate string, omitEmpty bool) string {
	jsonPart := jsonName
	if omitEmpty {
		jsonPart = jsonPart + ",omitempty"
	}
	parts := []string{fmt.Sprintf("json:\"%s\"", jsonPart)}
	if strings.TrimSpace(validate) != "" {
		parts = append(parts, fmt.Sprintf("validate:\"%s\"", validate))
	}
	return strings.Join(parts, " ")
}

func buildUpdateValidateTag(validate string) string {
	if strings.TrimSpace(validate) == "" {
		return "omitempty"
	}
	return "omitempty," + validate
}

func buildResponseTag(jsonName string, omitEmpty bool) string {
	if omitEmpty {
		return fmt.Sprintf("json:\"%s,omitempty\"", jsonName)
	}
	return fmt.Sprintf("json:\"%s\"", jsonName)
}

func buildCreateAssignments(fields []fieldInfo) []string {
	var assignments []string
	for _, f := range fields {
		if !f.IncludeInCreate {
			continue
		}
		assignments = append(assignments, fmt.Sprintf("%s: req.%s,", f.Name, f.Name))
	}
	return assignments
}

func buildUpdateStatements(fields []fieldInfo) []string {
	var statements []string
	for _, f := range fields {
		if !f.IncludeInUpdate {
			continue
		}

		assignExpr := fmt.Sprintf("req.%s", f.Name)
		if !f.IsPointer {
			assignExpr = fmt.Sprintf("*req.%s", f.Name)
		}
		statements = append(statements, fmt.Sprintf("if req.%s != nil {\n\t\tentity.%s = %s\n\t}", f.Name, f.Name, assignExpr))
	}
	return statements
}

func buildResponseMappings(fields []fieldInfo) []string {
	var mappings []string
	for _, f := range fields {
		if !f.IncludeInResp {
			continue
		}
		mappings = append(mappings, fmt.Sprintf("%s: entity.%s,", f.Name, f.Name))
	}
	return mappings
}

func buildDTOImports(fields []fieldInfo, imports map[string]string) []importSpec {
	set := map[string]importSpec{}
	for _, f := range fields {
		if f.IncludeInCreate || f.IncludeInUpdate || f.IncludeInResp {
			for _, alias := range f.Aliases {
				path := resolveImportPath(alias, imports)
				if path != "" {
					set[path] = importSpec{Alias: aliasIfNeeded(alias, path), Path: path}
				}
			}
		}
	}
	return mapToSlice(set)
}

func buildRepositoryImports() []importSpec {
	return []importSpec{
		{Path: "context"},
		{Path: "errors"},
		{Path: "gorm.io/gorm"},
	}
}

func buildServiceImports() []importSpec {
	return []importSpec{
		{Path: "context"},
		{Path: "errors"},
		{Path: "gorm.io/gorm"},
	}
}

func buildControllerImports() []importSpec {
	return []importSpec{
		{Path: "errors"},
		{Path: "net/http"},
		{Path: "github.com/gin-gonic/gin"},
		{Path: "go-base/internal/validation"},
		{Path: "go-base/pkg/response"},
	}
}

func defaultAliasForPath(path string) string {
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		return path[idx+1:]
	}
	return path
}

func resolveImportPath(alias string, imports map[string]string) string {
	if path, ok := imports[alias]; ok {
		return path
	}
	if alias == "time" {
		return "time"
	}
	return ""
}

func aliasIfNeeded(alias, path string) string {
	if alias == "" {
		return ""
	}
	if alias == defaultAliasForPath(path) {
		return ""
	}
	return alias
}

func mapToSlice(m map[string]importSpec) []importSpec {
	imports := make([]importSpec, 0, len(m))
	for _, v := range m {
		imports = append(imports, v)
	}
	sort.Slice(imports, func(i, j int) bool { return imports[i].Path < imports[j].Path })
	return imports
}

func uniqueImports(imports []importSpec) []importSpec {
	set := map[string]importSpec{}
	for _, imp := range imports {
		if imp.Path == "" {
			continue
		}
		set[imp.Path] = imp
	}
	return mapToSlice(set)
}

func buildIDSpec(idField fieldInfo, imports map[string]string) idSpec {
	idType := strings.TrimPrefix(idField.Type, "*")
	spec := idSpec{Type: idType, VarName: "id", ZeroValue: zeroValueForType(idType)}

	switch idType {
	case "string":
		spec.ParseCode = "\tid := c.Param(\"id\")"
	case "uuid.UUID":
		path := resolveImportPath("uuid", imports)
		if path == "" {
			path = "github.com/google/uuid"
		}
		spec.ExtraImports = append(spec.ExtraImports, importSpec{Alias: aliasIfNeeded("uuid", path), Path: path})
		spec.ParseCode = "\tidParam := c.Param(\"id\")\n\tuuidValue, err := uuid.Parse(idParam)\n\tif err != nil {\n\t\tstatus, payload := response.Fail(\"invalid_id\", \"Invalid identifier supplied\", err.Error())\n\t\tc.AbortWithStatusJSON(status, payload)\n\t\treturn\n\t}\n\tid := uuidValue"
	default:
		if strings.HasPrefix(idType, "int") {
			spec.ExtraImports = append(spec.ExtraImports, importSpec{Path: "strconv"})
			spec.ParseCode = fmt.Sprintf("\tidValue, err := strconv.ParseInt(c.Param(\"id\"), 10, %s)\n\tif err != nil {\n\t\tstatus, payload := response.Fail(\"invalid_id\", \"Invalid identifier supplied\", err.Error())\n\t\tc.AbortWithStatusJSON(status, payload)\n\t\treturn\n\t}\n\tid := %s(idValue)", bitSizeForType(idType), idType)
		} else if strings.HasPrefix(idType, "uint") {
			spec.ExtraImports = append(spec.ExtraImports, importSpec{Path: "strconv"})
			spec.ParseCode = fmt.Sprintf("\tidValue, err := strconv.ParseUint(c.Param(\"id\"), 10, %s)\n\tif err != nil {\n\t\tstatus, payload := response.Fail(\"invalid_id\", \"Invalid identifier supplied\", err.Error())\n\t\tc.AbortWithStatusJSON(status, payload)\n\t\treturn\n\t}\n\tid := %s(idValue)", bitSizeForType(idType), idType)
		} else {
			spec.ParseCode = "\tid := c.Param(\"id\")"
		}
	}

	spec.ParseCode += "\n"
	return spec
}

func bitSizeForType(t string) string {
	switch t {
	case "int8", "uint8":
		return "8"
	case "int16", "uint16":
		return "16"
	case "int32", "uint32":
		return "32"
	default:
		return "64"
	}
}

func zeroValueForType(t string) string {
	switch t {
	case "string":
		return "\"\""
	case "bool":
		return "false"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return "0"
	default:
		return t + "{}"
	}
}

const dtoTemplate = `package {{ .Package }}

import (
{{- range .DTOImports }}
    {{- if .Alias }}{{ .Alias }} {{ end }}"{{ .Path }}"
{{- end }}
)

type Create{{ .Entity }}Request struct {
{{- range .CreateFields }}
    {{ .Name }} {{ .Type }} {{ backtick .Tag }}
{{- end }}
}

type Update{{ .Entity }}Request struct {
{{- range .UpdateFields }}
    {{ .Name }} {{ .Type }} {{ backtick .Tag }}
{{- end }}
}

type {{ .Entity }}Response struct {
{{- range .ResponseFields }}
    {{ .Name }} {{ .Type }} {{ backtick .Tag }}
{{- end }}
}

func New{{ .Entity }}Response(entity *{{ .Entity }}) {{ .Entity }}Response {
    return {{ .Entity }}Response{
{{- range .ResponseMappings }}
        {{ . }}
{{- end }}
    }
}

func New{{ .Entity }}Responses(entities []{{ .Entity }}) []{{ .Entity }}Response {
    responses := make([]{{ .Entity }}Response, len(entities))
    for i := range entities {
        responses[i] = New{{ .Entity }}Response(&entities[i])
    }
    return responses
}
`

const repositoryTemplate = `package {{ .Package }}

import (
{{- range .RepositoryImports }}
    {{- if .Alias }}{{ .Alias }} {{ end }}"{{ .Path }}"
{{- end }}
)

type {{ .Entity }}Repository interface {
    Create(ctx context.Context, entity *{{ .Entity }}) error
    FindByID(ctx context.Context, id {{ .IDSpec.Type }}) (*{{ .Entity }}, error)
    List(ctx context.Context) ([]{{ .Entity }}, error)
    Update(ctx context.Context, entity *{{ .Entity }}) error
    Delete(ctx context.Context, id {{ .IDSpec.Type }}) error
}

type {{ .EntityLower }}Repository struct {
    db *gorm.DB
}

func New{{ .Entity }}Repository(db *gorm.DB) {{ .Entity }}Repository {
    return &{{ .EntityLower }}Repository{db: db}
}

func (r *{{ .EntityLower }}Repository) Create(ctx context.Context, entity *{{ .Entity }}) error {
    return r.db.WithContext(ctx).Create(entity).Error
}

func (r *{{ .EntityLower }}Repository) FindByID(ctx context.Context, id {{ .IDSpec.Type }}) (*{{ .Entity }}, error) {
    var entity {{ .Entity }}
    if err := r.db.WithContext(ctx).Where("id = ?", id).First(&entity).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, err
        }
        return nil, err
    }
    return &entity, nil
}

func (r *{{ .EntityLower }}Repository) List(ctx context.Context) ([]{{ .Entity }}, error) {
    var entities []{{ .Entity }}
    if err := r.db.WithContext(ctx).Find(&entities).Error; err != nil {
        return nil, err
    }
    return entities, nil
}

func (r *{{ .EntityLower }}Repository) Update(ctx context.Context, entity *{{ .Entity }}) error {
    return r.db.WithContext(ctx).Save(entity).Error
}

func (r *{{ .EntityLower }}Repository) Delete(ctx context.Context, id {{ .IDSpec.Type }}) error {
    return r.db.WithContext(ctx).Where("id = ?", id).Delete(&{{ .Entity }}{}).Error
}
`

const serviceTemplate = `package {{ .Package }}

import (
{{- range .ServiceImports }}
    {{- if .Alias }}{{ .Alias }} {{ end }}"{{ .Path }}"
{{- end }}
)

var Err{{ .Entity }}NotFound = errors.New("{{ .EntitySnake }} not found")

type {{ .Entity }}Service interface {
    Create(ctx context.Context, req Create{{ .Entity }}Request) (*{{ .Entity }}, error)
    Get(ctx context.Context, id {{ .IDSpec.Type }}) (*{{ .Entity }}, error)
    List(ctx context.Context) ([]{{ .Entity }}, error)
    Update(ctx context.Context, id {{ .IDSpec.Type }}, req Update{{ .Entity }}Request) (*{{ .Entity }}, error)
    Delete(ctx context.Context, id {{ .IDSpec.Type }}) error
}

type {{ .EntityLower }}Service struct {
    repo {{ .Entity }}Repository
}

func New{{ .Entity }}Service(repo {{ .Entity }}Repository) {{ .Entity }}Service {
    return &{{ .EntityLower }}Service{repo: repo}
}

func (s *{{ .EntityLower }}Service) Create(ctx context.Context, req Create{{ .Entity }}Request) (*{{ .Entity }}, error) {
    entity := &{{ .Entity }}{
{{- range .CreateAssignments }}
        {{ . }}
{{- end }}
    }
    if err := s.repo.Create(ctx, entity); err != nil {
        return nil, err
    }
    return entity, nil
}

func (s *{{ .EntityLower }}Service) Get(ctx context.Context, id {{ .IDSpec.Type }}) (*{{ .Entity }}, error) {
    entity, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, Err{{ .Entity }}NotFound
        }
        return nil, err
    }
    return entity, nil
}

func (s *{{ .EntityLower }}Service) List(ctx context.Context) ([]{{ .Entity }}, error) {
    return s.repo.List(ctx)
}

func (s *{{ .EntityLower }}Service) Update(ctx context.Context, id {{ .IDSpec.Type }}, req Update{{ .Entity }}Request) (*{{ .Entity }}, error) {
    entity, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, Err{{ .Entity }}NotFound
        }
        return nil, err
    }

{{- range .UpdateStatements }}
    {{ . }}
{{- end }}

    if err := s.repo.Update(ctx, entity); err != nil {
        return nil, err
    }
    return entity, nil
}

func (s *{{ .EntityLower }}Service) Delete(ctx context.Context, id {{ .IDSpec.Type }}) error {
    if err := s.repo.Delete(ctx, id); err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return Err{{ .Entity }}NotFound
        }
        return err
    }
    return nil
}
`

const controllerTemplate = `package {{ .Package }}

import (
{{- range .ControllerImports }}
    {{- if .Alias }}{{ .Alias }} {{ end }}"{{ .Path }}"
{{- end }}
)

type {{ .Entity }}Controller struct {
    service   {{ .Entity }}Service
    validator validation.Validator
}

func New{{ .Entity }}Controller(service {{ .Entity }}Service, validator validation.Validator) *{{ .Entity }}Controller {
    return &{{ .Entity }}Controller{service: service, validator: validator}
}

func (h *{{ .Entity }}Controller) RegisterRoutes(r *gin.RouterGroup) {
    r.POST("", h.Create)
    r.GET("", h.List)
    r.GET("/:id", h.GetByID)
    r.PUT("/:id", h.Update)
    r.DELETE("/:id", h.Delete)
}

func (h *{{ .Entity }}Controller) Create(c *gin.Context) {
    var req Create{{ .Entity }}Request
    if err := c.ShouldBindJSON(&req); err != nil {
        status, payload := response.Fail("invalid_request", "Invalid payload", err.Error())
        c.AbortWithStatusJSON(status, payload)
        return
    }

    if err := h.validator.Struct(req); err != nil {
        status, payload := response.Fail("invalid_request", "Validation failed", err.Error())
        c.AbortWithStatusJSON(status, payload)
        return
    }

    entity, err := h.service.Create(c.Request.Context(), req)
    if err != nil {
        status, payload := response.Fail("create_failed", "Could not create resource", err.Error())
        c.AbortWithStatusJSON(status, payload)
        return
    }

    c.JSON(http.StatusCreated, response.JSON(New{{ .Entity }}Response(entity), nil))
}

func (h *{{ .Entity }}Controller) List(c *gin.Context) {
    entities, err := h.service.List(c.Request.Context())
    if err != nil {
        status, payload := response.Fail("list_failed", "Could not list resources", err.Error())
        c.AbortWithStatusJSON(status, payload)
        return
    }

    c.JSON(http.StatusOK, response.JSON(New{{ .Entity }}Responses(entities), nil))
}

func (h *{{ .Entity }}Controller) GetByID(c *gin.Context) {
{{ .IDSpec.ParseCode }}
    entity, err := h.service.Get(c.Request.Context(), id)
    if err != nil {
        if errors.Is(err, Err{{ .Entity }}NotFound) {
            status, payload := response.WithStatus(http.StatusNotFound, "not_found", "Resource not found", nil)
            c.AbortWithStatusJSON(status, payload)
            return
        }
        status, payload := response.Fail("fetch_failed", "Could not fetch resource", err.Error())
        c.AbortWithStatusJSON(status, payload)
        return
    }

    c.JSON(http.StatusOK, response.JSON(New{{ .Entity }}Response(entity), nil))
}

func (h *{{ .Entity }}Controller) Update(c *gin.Context) {
{{ .IDSpec.ParseCode }}
    var req Update{{ .Entity }}Request
    if err := c.ShouldBindJSON(&req); err != nil {
        status, payload := response.Fail("invalid_request", "Invalid payload", err.Error())
        c.AbortWithStatusJSON(status, payload)
        return
    }

    if err := h.validator.Struct(req); err != nil {
        status, payload := response.Fail("invalid_request", "Validation failed", err.Error())
        c.AbortWithStatusJSON(status, payload)
        return
    }

    entity, err := h.service.Update(c.Request.Context(), id, req)
    if err != nil {
        if errors.Is(err, Err{{ .Entity }}NotFound) {
            status, payload := response.WithStatus(http.StatusNotFound, "not_found", "Resource not found", nil)
            c.AbortWithStatusJSON(status, payload)
            return
        }
        status, payload := response.Fail("update_failed", "Could not update resource", err.Error())
        c.AbortWithStatusJSON(status, payload)
        return
    }

    c.JSON(http.StatusOK, response.JSON(New{{ .Entity }}Response(entity), nil))
}

func (h *{{ .Entity }}Controller) Delete(c *gin.Context) {
{{ .IDSpec.ParseCode }}
    if err := h.service.Delete(c.Request.Context(), id); err != nil {
        if errors.Is(err, Err{{ .Entity }}NotFound) {
            status, payload := response.WithStatus(http.StatusNotFound, "not_found", "Resource not found", nil)
            c.AbortWithStatusJSON(status, payload)
            return
        }
        status, payload := response.Fail("delete_failed", "Could not delete resource", err.Error())
        c.AbortWithStatusJSON(status, payload)
        return
    }

		c.Status(http.StatusNoContent)
}
`

const moduleTemplate = `package {{ .Package }}

import (
{{- range .ModuleImports }}
    {{- if .Alias }}{{ .Alias }} {{ end }}"{{ .Path }}"
{{- end }}
)

func init() {
    modules.Register(func(router *gin.RouterGroup, deps modules.Dependencies) {
        repo := New{{ .Entity }}Repository(deps.DB.DB())
        service := New{{ .Entity }}Service(repo)
        controller := New{{ .Entity }}Controller(service, deps.Validator)

        group := router.Group("/{{ .EntityPlural }}")
        group.Use(middleware.Auth(deps.JWT))
        controller.RegisterRoutes(group)
    })
}
`

package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/dukedhal/taskflow/internal/model"
	"github.com/dukedhal/taskflow/internal/service"
)

var validate = validator.New()

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var in service.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if fields := validateStruct(in); fields != nil {
		ValidationError(w, fields)
		return
	}

	resp, err := h.auth.Register(r.Context(), in)
	if errors.Is(err, model.ErrConflict) {
		Error(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	JSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in service.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if fields := validateStruct(in); fields != nil {
		ValidationError(w, fields)
		return
	}

	resp, err := h.auth.Login(r.Context(), in)
	if errors.Is(err, model.ErrUnauthorized) {
		Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	JSON(w, http.StatusOK, resp)
}

// validateStruct runs go-playground/validator and returns a field-keyed error
// map, or nil if validation passed.
func validateStruct(v any) map[string]string {
	err := validate.Struct(v)
	if err == nil {
		return nil
	}

	fields := make(map[string]string)
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, fe := range ve {
			fields[toSnakeCase(fe.Field())] = validationMessage(fe)
		}
	}
	return fields
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "is too short (min " + fe.Param() + " characters)"
	case "max":
		return "is too long (max " + fe.Param() + " characters)"
	case "oneof":
		return "must be one of: " + fe.Param()
	case "datetime":
		return "must be a valid date (YYYY-MM-DD)"
	default:
		return "is invalid"
	}
}

// toSnakeCase converts a PascalCase field name to snake_case for JSON field keys.
func toSnakeCase(s string) string {
	var result []byte
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c+32))
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}

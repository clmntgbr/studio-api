package handler

import (
	"strings"

	cmdproject "go-api/internal/application/command/project"
	queryproject "go-api/internal/application/query/project"
	"go-api/internal/domain/paginate"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type ProjectHandler struct {
	createHandler projectCreateHandler
	updateHandler projectUpdateHandler
	deleteHandler projectDeleteHandler
	getHandler    projectGetHandler
	listHandler   projectListHandler
}

func NewProjectHandler(
	createHandler projectCreateHandler,
	updateHandler projectUpdateHandler,
	deleteHandler projectDeleteHandler,
	getHandler projectGetHandler,
	listHandler projectListHandler,
) *ProjectHandler {
	return &ProjectHandler{
		createHandler: createHandler,
		updateHandler: updateHandler,
		deleteHandler: deleteHandler,
		getHandler:    getHandler,
		listHandler:   listHandler,
	}
}

type createProjectBody struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

type updateProjectBody struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

func (h *ProjectHandler) Create(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	var body createProjectBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	p, err := h.createHandler.Handle(c.Context(), cmdproject.CreateProjectCommand{Name: body.Name})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create project"})
	}
	return c.Status(fiber.StatusCreated).JSON(presenter.NewProjectResponseFromEntity(p))
}

func (h *ProjectHandler) List(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	var listQuery paginate.PaginateQuery
	if err := c.Bind().Query(&listQuery); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid query parameters",
			"errors":  err.Error(),
		})
	}
	sortBy, orderBy := listQuery.SortBy, listQuery.OrderBy
	listQuery.Normalize()
	if sortBy == "" {
		listQuery.SortBy = "created_at"
	}
	if orderBy == "" {
		listQuery.OrderBy = paginate.OrderByDesc
	}

	views, total, err := h.listHandler.Handle(c.Context(), queryproject.ListProjectsQuery{Query: listQuery})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list projects"})
	}
	return c.Status(fiber.StatusOK).JSON(paginate.NewPaginateResponse(
		presenter.NewProjectListResponseFromViews(views),
		int(total),
		listQuery,
	))
}

func (h *ProjectHandler) GetByID(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	view, err := h.getHandler.Handle(c.Context(), queryproject.GetProjectQuery{ID: id})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Project not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get project"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewProjectResponseFromView(*view))
}

func (h *ProjectHandler) Update(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	var body updateProjectBody
	if err := validation.BindBody(c, &body); err != nil {
		return err
	}
	p, err := h.updateHandler.Handle(c.Context(), cmdproject.UpdateProjectCommand{ID: id, Name: body.Name})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Project not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update project"})
	}
	return c.Status(fiber.StatusOK).JSON(presenter.NewProjectResponseFromEntity(p))
}

func (h *ProjectHandler) Delete(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid project id"})
	}
	if err := h.deleteHandler.Handle(c.Context(), cmdproject.DeleteProjectCommand{ID: id}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Project not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to delete project"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

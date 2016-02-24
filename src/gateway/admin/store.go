package admin

import (
	"errors"
	"io"
	"net/http"

	"gateway/config"
	aphttp "gateway/http"
	"gateway/logreport"
	apsql "gateway/sql"
	"gateway/store"

	"github.com/gorilla/handlers"
)

type StoreController interface {
	List(w http.ResponseWriter, r *http.Request) aphttp.Error
	Create(w http.ResponseWriter, r *http.Request) aphttp.Error
	Show(w http.ResponseWriter, r *http.Request) aphttp.Error
	Update(w http.ResponseWriter, r *http.Request) aphttp.Error
	Delete(w http.ResponseWriter, r *http.Request) aphttp.Error
}

func RouteStoreResource(controller StoreController, path string,
	router aphttp.Router, conf config.ProxyAdmin) {

	collectionRoutes := map[string]http.Handler{
		"GET":  process(controller.List),
		"POST": process(controller.Create),
	}
	instanceRoutes := map[string]http.Handler{
		"GET":    process(controller.Show),
		"PUT":    process(controller.Update),
		"DELETE": process(controller.Delete),
	}

	if conf.CORSEnabled {
		collectionRoutes["OPTIONS"] = aphttp.CORSOptionsHandler([]string{"GET", "POST", "OPTIONS"})
		instanceRoutes["OPTIONS"] = aphttp.CORSOptionsHandler([]string{"GET", "PUT", "DELETE", "OPTIONS"})
	}

	router.Handle(path, handlers.MethodHandler(collectionRoutes))
	router.Handle(path+"/{id}",
		handlers.HTTPMethodOverrideHandler(handlers.MethodHandler(instanceRoutes)))
}

func process(handler aphttp.ErrorReturningHandler) http.Handler {
	return aphttp.JSONErrorCatchingHandler(handler)
}

type StoreCollectionsController struct {
	BaseController
	store store.Store
}

func (c *StoreCollectionsController) List(w http.ResponseWriter, r *http.Request) aphttp.Error {
	collection := &store.Collection{AccountID: c.accountID(r)}
	var collections []*store.Collection
	err := c.store.ListCollection(collection, &collections)

	if err != nil {
		logreport.Printf("%s Error listing collection: %v\n%v", config.System, err, r)
		return aphttp.DefaultServerError()
	}

	return c.serializeCollection(collections, w)
}

func (c *StoreCollectionsController) Create(w http.ResponseWriter, r *http.Request) aphttp.Error {
	collection, httpErr := c.deserializeInstance(r.Body)
	if httpErr != nil {
		return httpErr
	}

	collection.AccountID = c.accountID(r)

	validationErrors := collection.Validate()
	if !validationErrors.Empty() {
		return SerializableValidationErrors{validationErrors}
	}

	err := c.store.CreateCollection(collection)
	if err != nil {
		if err == apsql.ErrZeroRowsAffected {
			return c.notFound()
		}
		logreport.Printf("%s Error inserting collection: %v\n%v", config.System, err, r)
		return aphttp.NewServerError(err)
	}

	return c.serializeInstance(collection, w)
}

func (c *StoreCollectionsController) Show(w http.ResponseWriter, r *http.Request) aphttp.Error {
	collection := &store.Collection{ID: instanceID(r), AccountID: c.accountID(r)}
	err := c.store.ShowCollection(collection)
	if err != nil {
		return c.notFound()
	}

	return c.serializeInstance(collection, w)
}

func (c *StoreCollectionsController) Update(w http.ResponseWriter, r *http.Request) aphttp.Error {
	collection, httpErr := c.deserializeInstance(r.Body)
	if httpErr != nil {
		return httpErr
	}

	collection.ID = instanceID(r)
	collection.AccountID = c.accountID(r)

	validationErrors := collection.Validate()
	if !validationErrors.Empty() {
		return SerializableValidationErrors{validationErrors}
	}

	err := c.store.UpdateCollection(collection)
	if err != nil {
		if err == apsql.ErrZeroRowsAffected {
			return c.notFound()
		}
		logreport.Printf("%s Error updating collection: %v\n%v", config.System, err, r)
		return aphttp.NewServerError(err)
	}

	return c.serializeInstance(collection, w)
}

func (c *StoreCollectionsController) Delete(w http.ResponseWriter, r *http.Request) aphttp.Error {
	collection := &store.Collection{ID: instanceID(r), AccountID: c.accountID(r)}
	err := c.store.DeleteCollection(collection)

	if err != nil {
		if err == apsql.ErrZeroRowsAffected {
			return c.notFound()
		}
		logreport.Printf("%s Error deleting collection: %v\n%v", config.System, err, r)
		return aphttp.NewServerError(err)
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func (c *StoreCollectionsController) notFound() aphttp.Error {
	return aphttp.NewError(errors.New("No collection matches"), 404)
}

func (c *StoreCollectionsController) deserializeInstance(file io.Reader) (*store.Collection,
	aphttp.Error) {

	var wrapped struct {
		Collection *store.Collection `json:"collection"`
	}
	if err := deserialize(&wrapped, file); err != nil {
		return nil, err
	}
	if wrapped.Collection == nil {
		return nil, aphttp.NewError(errors.New("Could not deserialize Collection from JSON."),
			http.StatusBadRequest)
	}
	return wrapped.Collection, nil
}

func (c *StoreCollectionsController) serializeInstance(instance *store.Collection,
	w http.ResponseWriter) aphttp.Error {

	wrapped := struct {
		Collection *store.Collection `json:"collection"`
	}{instance}
	return serialize(wrapped, w)
}

func (c *StoreCollectionsController) serializeCollection(collection []*store.Collection,
	w http.ResponseWriter) aphttp.Error {

	wrapped := struct {
		Collections []*store.Collection `json:"collections"`
	}{collection}
	return serialize(wrapped, w)
}

type StoreObjectsController struct {
	BaseController
	store store.Store
}

func (c *StoreObjectsController) List(w http.ResponseWriter, r *http.Request) aphttp.Error {
	object := &store.Object{AccountID: c.accountID(r), CollectionID: c.collectionID(r)}
	var objects []*store.Object
	err := c.store.ListObject(object, &objects)

	if err != nil {
		logreport.Printf("%s Error listing object: %v\n%v", config.System, err, r)
		return aphttp.DefaultServerError()
	}

	return c.serializeCollection(objects, w)
}

func (c *StoreObjectsController) Create(w http.ResponseWriter, r *http.Request) aphttp.Error {
	object, httpErr := c.deserializeInstance(r.Body)
	if httpErr != nil {
		return httpErr
	}

	object.AccountID = c.accountID(r)
	object.CollectionID = c.collectionID(r)

	validationErrors := object.Validate()
	if !validationErrors.Empty() {
		return SerializableValidationErrors{validationErrors}
	}

	err := c.store.CreateObject(object)
	if err != nil {
		if err == apsql.ErrZeroRowsAffected {
			return c.notFound()
		}
		logreport.Printf("%s Error inserting object: %v\n%v", config.System, err, r)
		return aphttp.NewServerError(err)
	}

	return c.serializeInstance(object, w)
}

func (c *StoreObjectsController) Show(w http.ResponseWriter, r *http.Request) aphttp.Error {
	object := &store.Object{
		ID:           instanceID(r),
		AccountID:    c.accountID(r),
		CollectionID: c.collectionID(r),
	}
	err := c.store.ShowObject(object)
	if err != nil {
		return c.notFound()
	}

	return c.serializeInstance(object, w)
}

func (c *StoreObjectsController) Update(w http.ResponseWriter, r *http.Request) aphttp.Error {
	object, httpErr := c.deserializeInstance(r.Body)
	if httpErr != nil {
		return httpErr
	}

	object.ID = instanceID(r)
	object.AccountID = c.accountID(r)
	object.CollectionID = c.collectionID(r)

	validationErrors := object.Validate()
	if !validationErrors.Empty() {
		return SerializableValidationErrors{validationErrors}
	}

	err := c.store.UpdateObject(object)
	if err != nil {
		if err == apsql.ErrZeroRowsAffected {
			return c.notFound()
		}
		logreport.Printf("%s Error updating object: %v\n%v", config.System, err, r)
		return aphttp.NewServerError(err)
	}

	return c.serializeInstance(object, w)
}

func (c *StoreObjectsController) Delete(w http.ResponseWriter, r *http.Request) aphttp.Error {
	object := &store.Object{
		ID:           instanceID(r),
		AccountID:    c.accountID(r),
		CollectionID: c.collectionID(r),
	}
	err := c.store.DeleteObject(object)

	if err != nil {
		if err == apsql.ErrZeroRowsAffected {
			return c.notFound()
		}
		logreport.Printf("%s Error deleting object: %v\n%v", config.System, err, r)
		return aphttp.NewServerError(err)
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func (c *StoreObjectsController) notFound() aphttp.Error {
	return aphttp.NewError(errors.New("No object matches"), 404)
}

func (c *StoreObjectsController) deserializeInstance(file io.Reader) (*store.Object,
	aphttp.Error) {

	var wrapped struct {
		Object *store.Object `json:"object"`
	}
	if err := deserialize(&wrapped, file); err != nil {
		return nil, err
	}
	if wrapped.Object == nil {
		return nil, aphttp.NewError(errors.New("Could not deserialize Object from JSON."),
			http.StatusBadRequest)
	}
	return wrapped.Object, nil
}

func (c *StoreObjectsController) serializeInstance(instance *store.Object,
	w http.ResponseWriter) aphttp.Error {

	wrapped := struct {
		Object *store.Object `json:"object"`
	}{instance}
	return serialize(wrapped, w)
}

func (c *StoreObjectsController) serializeCollection(collection []*store.Object,
	w http.ResponseWriter) aphttp.Error {

	wrapped := struct {
		Objects []*store.Object `json:"objects"`
	}{collection}
	return serialize(wrapped, w)
}

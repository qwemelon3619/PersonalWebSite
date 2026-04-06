package service

import (
	"errors"
	"testing"
	"time"

	"seungpyo.lee/PersonalWebSite/services/post-service/internal/adapter"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/model"
)

type stubPostRepo struct {
	createFn func(post *domain.Post) error
	getByID  func(id uint) (*domain.Post, error)
	getAll   func(filter model.PostFilter) ([]*domain.Post, error)
	updateFn func(post *domain.Post) error
	deleteFn func(id uint) error
}

func (s *stubPostRepo) Create(post *domain.Post) error        { return s.createFn(post) }
func (s *stubPostRepo) GetByID(id uint) (*domain.Post, error) { return s.getByID(id) }
func (s *stubPostRepo) GetAll(filter model.PostFilter) ([]*domain.Post, error) {
	return s.getAll(filter)
}
func (s *stubPostRepo) Update(post *domain.Post) error                      { return s.updateFn(post) }
func (s *stubPostRepo) Delete(id uint) error                                { return s.deleteFn(id) }
func (s *stubPostRepo) GetByAuthorID(authorID uint) ([]*domain.Post, error) { return nil, nil }

type stubTagRepo struct {
	attachFn     func(postID uint, tagNames []string) error
	replaceFn    func(postID uint, tagNames []string) error
	getTagsFn    func(postID uint) ([]*domain.Tag, error)
	listTagsFn   func() ([]*domain.Tag, error)
	deleteUnused func(tagID uint) error
}

func (s *stubTagRepo) AttachTagsToPost(postID uint, tagNames []string) error {
	return s.attachFn(postID, tagNames)
}
func (s *stubTagRepo) ReplaceTagsForPost(postID uint, tagNames []string) error {
	return s.replaceFn(postID, tagNames)
}
func (s *stubTagRepo) GetTagsForPost(postID uint) ([]*domain.Tag, error) { return s.getTagsFn(postID) }
func (s *stubTagRepo) ListTags() ([]*domain.Tag, error)                  { return s.listTagsFn() }
func (s *stubTagRepo) DeleteTag(id uint) error                           { return nil }
func (s *stubTagRepo) DeleteUnusedTag(tagID uint) error                  { return s.deleteUnused(tagID) }

type stubImageAdapter struct {
	processFn func(content string, userID uint) (string, error)
	uploadFn  func(data string, userID uint) (string, error)
	deleteFn  func(path string) error
	extractFn func(content string) []string
}

func (s *stubImageAdapter) UploadImage(data string, userID uint) (string, error) {
	return s.uploadFn(data, userID)
}
func (s *stubImageAdapter) DeleteImage(path string) error { return s.deleteFn(path) }
func (s *stubImageAdapter) ProcessMarkdownForImages(content string, userID uint) (string, error) {
	return s.processFn(content, userID)
}
func (s *stubImageAdapter) ExtractImageURLsFromContent(content string) []string {
	return s.extractFn(content)
}

type stubTranslationAdapter struct {
	singleFn   func(text string) (string, error)
	markdownFn func(content string) (string, error)
}

func (s *stubTranslationAdapter) TranslateSingle(text string) (string, error) {
	return s.singleFn(text)
}
func (s *stubTranslationAdapter) TranslateMarkdown(content string) (string, error) {
	return s.markdownFn(content)
}

func newSvcForTest(postRepo domain.PostRepository, tagRepo domain.TagRepository, cfg *config.PostConfig, img adapter.ImageAdapter, tr adapter.TranslationAdapter) *postService {
	return NewPostService(postRepo, tagRepo, cfg, img, tr).(*postService)
}

func TestCreatePost_Flow(t *testing.T) {
	asyncCalled := make(chan struct{}, 1)
	post := &domain.Post{ID: 1, AuthorID: 7}
	svc := newSvcForTest(
		&stubPostRepo{
			createFn: func(p *domain.Post) error { p.ID = 1; return nil },
			getByID:  func(id uint) (*domain.Post, error) { return post, nil },
			getAll:   func(filter model.PostFilter) ([]*domain.Post, error) { return nil, nil },
			updateFn: func(post *domain.Post) error { return nil },
			deleteFn: func(id uint) error { return nil },
		},
		&stubTagRepo{
			attachFn:     func(postID uint, tagNames []string) error { return nil },
			replaceFn:    func(postID uint, tagNames []string) error { return nil },
			getTagsFn:    func(postID uint) ([]*domain.Tag, error) { return nil, nil },
			listTagsFn:   func() ([]*domain.Tag, error) { return nil, nil },
			deleteUnused: func(tagID uint) error { return nil },
		},
		&config.PostConfig{TranslationAPIURL: "http://translate"},
		&stubImageAdapter{
			processFn: func(content string, userID uint) (string, error) { return "processed", nil },
			uploadFn:  func(data string, userID uint) (string, error) { return "/thumb.png", nil },
			deleteFn:  func(path string) error { return nil },
			extractFn: func(content string) []string { return nil },
		},
		&stubTranslationAdapter{
			singleFn: func(text string) (string, error) {
				asyncCalled <- struct{}{}
				return "en-title", nil
			},
			markdownFn: func(content string) (string, error) { return "<p>en</p>", nil },
		},
	)

	got, err := svc.CreatePost(model.CreatePostRequest{
		Title: "title", Content: "content", ThumbnailData: "data", Tags: []string{"go"}, Published: true,
	}, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 1 {
		t.Fatalf("expected loaded post")
	}
	select {
	case <-asyncCalled:
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("expected async translation to run")
	}
}

func TestCreatePost_ErrorCases(t *testing.T) {
	basePostRepo := &stubPostRepo{
		createFn: func(p *domain.Post) error { p.ID = 1; return nil },
		getByID:  func(id uint) (*domain.Post, error) { return &domain.Post{ID: id}, nil },
		getAll:   func(filter model.PostFilter) ([]*domain.Post, error) { return nil, nil },
		updateFn: func(post *domain.Post) error { return nil },
		deleteFn: func(id uint) error { return nil },
	}
	baseTags := &stubTagRepo{
		attachFn:     func(postID uint, tagNames []string) error { return nil },
		replaceFn:    func(postID uint, tagNames []string) error { return nil },
		getTagsFn:    func(postID uint) ([]*domain.Tag, error) { return nil, nil },
		listTagsFn:   func() ([]*domain.Tag, error) { return nil, nil },
		deleteUnused: func(tagID uint) error { return nil },
	}
	baseImg := &stubImageAdapter{
		processFn: func(content string, userID uint) (string, error) { return "processed", nil },
		uploadFn:  func(data string, userID uint) (string, error) { return "/thumb.png", nil },
		deleteFn:  func(path string) error { return nil },
		extractFn: func(content string) []string { return nil },
	}
	baseTr := &stubTranslationAdapter{
		singleFn:   func(text string) (string, error) { return "en", nil },
		markdownFn: func(content string) (string, error) { return "<p>en</p>", nil },
	}

	// process error
	svc1 := newSvcForTest(basePostRepo, baseTags, &config.PostConfig{}, &stubImageAdapter{
		processFn: func(content string, userID uint) (string, error) { return "", errors.New("process fail") },
		uploadFn:  baseImg.uploadFn, deleteFn: baseImg.deleteFn, extractFn: baseImg.extractFn,
	}, baseTr)
	if _, err := svc1.CreatePost(model.CreatePostRequest{Title: "t", Content: "c"}, 1); err == nil {
		t.Fatalf("expected process error")
	}

	// thumbnail upload error
	svc2 := newSvcForTest(basePostRepo, baseTags, &config.PostConfig{}, &stubImageAdapter{
		processFn: baseImg.processFn,
		uploadFn:  func(data string, userID uint) (string, error) { return "", errors.New("upload fail") },
		deleteFn:  baseImg.deleteFn, extractFn: baseImg.extractFn,
	}, baseTr)
	if _, err := svc2.CreatePost(model.CreatePostRequest{Title: "t", Content: "c", ThumbnailData: "d"}, 1); err == nil {
		t.Fatalf("expected thumbnail error")
	}

	// create fail
	svc3 := newSvcForTest(&stubPostRepo{
		createFn: func(p *domain.Post) error { return errors.New("create fail") },
		getByID:  basePostRepo.getByID, getAll: basePostRepo.getAll, updateFn: basePostRepo.updateFn, deleteFn: basePostRepo.deleteFn,
	}, baseTags, &config.PostConfig{}, baseImg, baseTr)
	if _, err := svc3.CreatePost(model.CreatePostRequest{Title: "t", Content: "c"}, 1); err == nil {
		t.Fatalf("expected create error")
	}

	// attach tags fail
	svc4 := newSvcForTest(basePostRepo, &stubTagRepo{
		attachFn:  func(postID uint, tagNames []string) error { return errors.New("tag fail") },
		replaceFn: baseTags.replaceFn, getTagsFn: baseTags.getTagsFn, listTagsFn: baseTags.listTagsFn, deleteUnused: baseTags.deleteUnused,
	}, &config.PostConfig{}, baseImg, baseTr)
	if _, err := svc4.CreatePost(model.CreatePostRequest{Title: "t", Content: "c", Tags: []string{"go"}}, 1); err == nil {
		t.Fatalf("expected attach tags error")
	}

	// load fail after create
	svc5 := newSvcForTest(&stubPostRepo{
		createFn: func(p *domain.Post) error { p.ID = 9; return nil },
		getByID:  func(id uint) (*domain.Post, error) { return nil, errors.New("load fail") },
		getAll:   basePostRepo.getAll, updateFn: basePostRepo.updateFn, deleteFn: basePostRepo.deleteFn,
	}, baseTags, &config.PostConfig{}, baseImg, baseTr)
	if _, err := svc5.CreatePost(model.CreatePostRequest{Title: "t", Content: "c"}, 1); err == nil {
		t.Fatalf("expected load fail")
	}
}

func TestCreatePost_TranslationDisabled_NoAsync(t *testing.T) {
	translated := false
	svc := newSvcForTest(
		&stubPostRepo{
			createFn: func(p *domain.Post) error { p.ID = 1; return nil },
			getByID:  func(id uint) (*domain.Post, error) { return &domain.Post{ID: id}, nil },
			getAll:   func(filter model.PostFilter) ([]*domain.Post, error) { return nil, nil },
			updateFn: func(post *domain.Post) error { return nil },
			deleteFn: func(id uint) error { return nil },
		},
		&stubTagRepo{
			attachFn:     func(postID uint, tagNames []string) error { return nil },
			replaceFn:    func(postID uint, tagNames []string) error { return nil },
			getTagsFn:    func(postID uint) ([]*domain.Tag, error) { return nil, nil },
			listTagsFn:   func() ([]*domain.Tag, error) { return nil, nil },
			deleteUnused: func(tagID uint) error { return nil },
		},
		&config.PostConfig{}, // URL empty => translation disabled
		&stubImageAdapter{
			processFn: func(content string, userID uint) (string, error) { return content, nil },
			uploadFn:  func(data string, userID uint) (string, error) { return "", nil },
			deleteFn:  func(path string) error { return nil },
			extractFn: func(content string) []string { return nil },
		},
		&stubTranslationAdapter{
			singleFn: func(text string) (string, error) { translated = true; return "en", nil },
			markdownFn: func(content string) (string, error) {
				translated = true
				return "<p>en</p>", nil
			},
		},
	)
	if _, err := svc.CreatePost(model.CreatePostRequest{Title: "title", Content: "content"}, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	time.Sleep(120 * time.Millisecond)
	if translated {
		t.Fatalf("translation should not run when disabled")
	}
}

func TestGetPostAndGetPostsByFilter(t *testing.T) {
	svc := newSvcForTest(
		&stubPostRepo{
			createFn: func(post *domain.Post) error { return nil },
			getByID:  func(id uint) (*domain.Post, error) { return &domain.Post{ID: id}, nil },
			getAll:   func(filter model.PostFilter) ([]*domain.Post, error) { return []*domain.Post{{ID: 1}}, nil },
			updateFn: func(post *domain.Post) error { return nil },
			deleteFn: func(id uint) error { return nil },
		},
		&stubTagRepo{
			attachFn:     func(postID uint, tagNames []string) error { return nil },
			replaceFn:    func(postID uint, tagNames []string) error { return nil },
			getTagsFn:    func(postID uint) ([]*domain.Tag, error) { return []*domain.Tag{{ID: 1, Name: "go"}}, nil },
			listTagsFn:   func() ([]*domain.Tag, error) { return []*domain.Tag{{ID: 1}}, nil },
			deleteUnused: func(tagID uint) error { return nil },
		},
		&config.PostConfig{},
		&stubImageAdapter{processFn: func(content string, userID uint) (string, error) { return content, nil }, uploadFn: func(data string, userID uint) (string, error) { return "", nil }, deleteFn: func(path string) error { return nil }, extractFn: func(content string) []string { return nil }},
		&stubTranslationAdapter{singleFn: func(text string) (string, error) { return "", nil }, markdownFn: func(content string) (string, error) { return "", nil }},
	)

	post, err := svc.GetPost(1)
	if err != nil || len(post.Tags) != 1 {
		t.Fatalf("expected post with tags, got %v err=%v", post, err)
	}
	posts, err := svc.GetPostsByFilter(model.PostFilter{})
	if err != nil || len(posts) != 1 {
		t.Fatalf("expected posts, got %v err=%v", posts, err)
	}
}

func TestGetPost_TagLoadFailureNonFatal(t *testing.T) {
	svc := newSvcForTest(
		&stubPostRepo{createFn: func(post *domain.Post) error { return nil }, getByID: func(id uint) (*domain.Post, error) { return &domain.Post{ID: id}, nil }, getAll: func(filter model.PostFilter) ([]*domain.Post, error) { return nil, nil }, updateFn: func(post *domain.Post) error { return nil }, deleteFn: func(id uint) error { return nil }},
		&stubTagRepo{attachFn: func(postID uint, tagNames []string) error { return nil }, replaceFn: func(postID uint, tagNames []string) error { return nil }, getTagsFn: func(postID uint) ([]*domain.Tag, error) { return nil, errors.New("tag fail") }, listTagsFn: func() ([]*domain.Tag, error) { return nil, nil }, deleteUnused: func(tagID uint) error { return nil }},
		&config.PostConfig{},
		&stubImageAdapter{processFn: func(content string, userID uint) (string, error) { return content, nil }, uploadFn: func(data string, userID uint) (string, error) { return "", nil }, deleteFn: func(path string) error { return nil }, extractFn: func(content string) []string { return nil }},
		&stubTranslationAdapter{singleFn: func(text string) (string, error) { return "", nil }, markdownFn: func(content string) (string, error) { return "", nil }},
	)
	post, err := svc.GetPost(1)
	if err != nil || post == nil {
		t.Fatalf("expected success despite tag error")
	}
}

func TestGetPostAndGetPostsByFilter_RepoErrors(t *testing.T) {
	svc := newSvcForTest(
		&stubPostRepo{
			createFn: func(post *domain.Post) error { return nil },
			getByID:  func(id uint) (*domain.Post, error) { return nil, errors.New("get fail") },
			getAll:   func(filter model.PostFilter) ([]*domain.Post, error) { return nil, errors.New("list fail") },
			updateFn: func(post *domain.Post) error { return nil },
			deleteFn: func(id uint) error { return nil },
		},
		&stubTagRepo{
			attachFn:     func(postID uint, tagNames []string) error { return nil },
			replaceFn:    func(postID uint, tagNames []string) error { return nil },
			getTagsFn:    func(postID uint) ([]*domain.Tag, error) { return nil, nil },
			listTagsFn:   func() ([]*domain.Tag, error) { return nil, nil },
			deleteUnused: func(tagID uint) error { return nil },
		},
		&config.PostConfig{},
		&stubImageAdapter{processFn: func(content string, userID uint) (string, error) { return content, nil }, uploadFn: func(data string, userID uint) (string, error) { return "", nil }, deleteFn: func(path string) error { return nil }, extractFn: func(content string) []string { return nil }},
		&stubTranslationAdapter{singleFn: func(text string) (string, error) { return "", nil }, markdownFn: func(content string) (string, error) { return "", nil }},
	)

	if _, err := svc.GetPost(1); err == nil {
		t.Fatalf("expected get post error")
	}
	if _, err := svc.GetPostsByFilter(model.PostFilter{}); err == nil {
		t.Fatalf("expected list posts error")
	}
}

func TestUpdatePost(t *testing.T) {
	title := "new"
	content := "new-content"
	thumb := "thumb-data"
	published := true
	tags := []string{"go"}
	translateCalled := make(chan struct{}, 1)

	postState := &domain.Post{ID: 1, AuthorID: 1, Title: "old", Content: "old"}
	svc := newSvcForTest(
		&stubPostRepo{
			createFn: func(post *domain.Post) error { return nil },
			getByID: func(id uint) (*domain.Post, error) {
				if id == 99 {
					return nil, errors.New("not found")
				}
				cp := *postState
				cp.ID = id
				return &cp, nil
			},
			getAll: func(filter model.PostFilter) ([]*domain.Post, error) { return nil, nil },
			updateFn: func(post *domain.Post) error {
				if post.ID == 2 {
					return errors.New("update fail")
				}
				return nil
			},
			deleteFn: func(id uint) error { return nil },
		},
		&stubTagRepo{
			attachFn: func(postID uint, tagNames []string) error { return nil },
			replaceFn: func(postID uint, tagNames []string) error {
				if postID == 3 {
					return errors.New("replace fail")
				}
				return nil
			},
			getTagsFn:    func(postID uint) ([]*domain.Tag, error) { return nil, nil },
			listTagsFn:   func() ([]*domain.Tag, error) { return nil, nil },
			deleteUnused: func(tagID uint) error { return nil },
		},
		&config.PostConfig{TranslationAPIURL: "on"},
		&stubImageAdapter{
			processFn: func(content string, userID uint) (string, error) {
				if content == "bad" {
					return "", errors.New("process fail")
				}
				return "processed", nil
			},
			uploadFn: func(data string, userID uint) (string, error) {
				if data == "bad-thumb" {
					return "", errors.New("upload fail")
				}
				return "/thumb.png", nil
			},
			deleteFn:  func(path string) error { return nil },
			extractFn: func(content string) []string { return nil },
		},
		&stubTranslationAdapter{
			singleFn: func(text string) (string, error) {
				translateCalled <- struct{}{}
				return "en", nil
			},
			markdownFn: func(content string) (string, error) { return "<p>en</p>", nil },
		},
	)

	if _, err := svc.UpdatePost(99, model.UpdatePostRequest{}, 1); err == nil {
		t.Fatalf("expected get post fail")
	}
	if _, err := svc.UpdatePost(1, model.UpdatePostRequest{}, 2); err == nil {
		t.Fatalf("expected unauthorized")
	}
	bad := "bad"
	if _, err := svc.UpdatePost(1, model.UpdatePostRequest{Content: &bad}, 1); err == nil {
		t.Fatalf("expected process fail")
	}
	badThumb := "bad-thumb"
	if _, err := svc.UpdatePost(1, model.UpdatePostRequest{ThumbnailData: &badThumb}, 1); err == nil {
		t.Fatalf("expected thumbnail fail")
	}
	if _, err := svc.UpdatePost(2, model.UpdatePostRequest{Title: &title}, 1); err == nil {
		t.Fatalf("expected update fail")
	}
	if _, err := svc.UpdatePost(3, model.UpdatePostRequest{Title: &title, Tags: &tags}, 1); err == nil {
		t.Fatalf("expected replace tags fail")
	}
	got, err := svc.UpdatePost(1, model.UpdatePostRequest{
		Title: &title, Content: &content, ThumbnailData: &thumb, Published: &published, Tags: &tags,
	}, 1)
	if err != nil || got.Title != "new" || got.Content != "processed" || got.Thumbnail != "/thumb.png" || !got.Published {
		t.Fatalf("unexpected update result %v err=%v", got, err)
	}
	select {
	case <-translateCalled:
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("expected async translation trigger")
	}
}

func TestUpdatePost_TranslationDisabled_NoAsync(t *testing.T) {
	title := "new"
	translated := false
	svc := newSvcForTest(
		&stubPostRepo{
			createFn: func(post *domain.Post) error { return nil },
			getByID: func(id uint) (*domain.Post, error) {
				return &domain.Post{ID: id, AuthorID: 1, Title: "old", Content: "old"}, nil
			},
			getAll:   func(filter model.PostFilter) ([]*domain.Post, error) { return nil, nil },
			updateFn: func(post *domain.Post) error { return nil },
			deleteFn: func(id uint) error { return nil },
		},
		&stubTagRepo{
			attachFn:     func(postID uint, tagNames []string) error { return nil },
			replaceFn:    func(postID uint, tagNames []string) error { return nil },
			getTagsFn:    func(postID uint) ([]*domain.Tag, error) { return nil, nil },
			listTagsFn:   func() ([]*domain.Tag, error) { return nil, nil },
			deleteUnused: func(tagID uint) error { return nil },
		},
		&config.PostConfig{},
		&stubImageAdapter{
			processFn: func(content string, userID uint) (string, error) { return content, nil },
			uploadFn:  func(data string, userID uint) (string, error) { return "", nil },
			deleteFn:  func(path string) error { return nil },
			extractFn: func(content string) []string { return nil },
		},
		&stubTranslationAdapter{
			singleFn:   func(text string) (string, error) { translated = true; return "en", nil },
			markdownFn: func(content string) (string, error) { translated = true; return "<p>en</p>", nil },
		},
	)
	if _, err := svc.UpdatePost(1, model.UpdatePostRequest{Title: &title}, 1); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	time.Sleep(120 * time.Millisecond)
	if translated {
		t.Fatalf("translation should not run when disabled")
	}
}

func TestUpdatePost_TitleOnly(t *testing.T) {
	title := "title-only"
	var updated *domain.Post
	svc := newSvcForTest(
		&stubPostRepo{
			createFn: func(post *domain.Post) error { return nil },
			getByID: func(id uint) (*domain.Post, error) {
				return &domain.Post{ID: id, AuthorID: 1, Title: "old", Content: "keep-content"}, nil
			},
			getAll:   func(filter model.PostFilter) ([]*domain.Post, error) { return nil, nil },
			updateFn: func(post *domain.Post) error { cp := *post; updated = &cp; return nil },
			deleteFn: func(id uint) error { return nil },
		},
		&stubTagRepo{
			attachFn:     func(postID uint, tagNames []string) error { return nil },
			replaceFn:    func(postID uint, tagNames []string) error { return nil },
			getTagsFn:    func(postID uint) ([]*domain.Tag, error) { return nil, nil },
			listTagsFn:   func() ([]*domain.Tag, error) { return nil, nil },
			deleteUnused: func(tagID uint) error { return nil },
		},
		&config.PostConfig{},
		&stubImageAdapter{
			processFn: func(content string, userID uint) (string, error) { return "processed", nil },
			uploadFn:  func(data string, userID uint) (string, error) { return "", nil },
			deleteFn:  func(path string) error { return nil },
			extractFn: func(content string) []string { return nil },
		},
		&stubTranslationAdapter{singleFn: func(text string) (string, error) { return "", nil }, markdownFn: func(content string) (string, error) { return "", nil }},
	)

	got, err := svc.UpdatePost(1, model.UpdatePostRequest{Title: &title}, 1)
	if err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if got.Title != "title-only" || got.Content != "keep-content" {
		t.Fatalf("unexpected updated post: %+v", got)
	}
	if updated == nil || updated.Title != "title-only" || updated.Content != "keep-content" {
		t.Fatalf("unexpected repo update payload: %+v", updated)
	}
}

func TestDeletePost(t *testing.T) {
	post := &domain.Post{
		ID: 1, AuthorID: 1, Thumbnail: "/thumb.png", Content: "![a](/a.png)",
		Tags: []*domain.Tag{{ID: 1, Name: "go"}},
	}
	deleteImageCalls := 0
	svc := newSvcForTest(
		&stubPostRepo{
			createFn: func(post *domain.Post) error { return nil },
			getByID: func(id uint) (*domain.Post, error) {
				if id == 99 {
					return nil, errors.New("not found")
				}
				return post, nil
			},
			getAll:   func(filter model.PostFilter) ([]*domain.Post, error) { return nil, nil },
			updateFn: func(post *domain.Post) error { return nil },
			deleteFn: func(id uint) error {
				if id == 2 {
					return errors.New("delete fail")
				}
				return nil
			},
		},
		&stubTagRepo{
			attachFn:     func(postID uint, tagNames []string) error { return nil },
			replaceFn:    func(postID uint, tagNames []string) error { return nil },
			getTagsFn:    func(postID uint) ([]*domain.Tag, error) { return nil, nil },
			listTagsFn:   func() ([]*domain.Tag, error) { return nil, nil },
			deleteUnused: func(tagID uint) error { return errors.New("tag cleanup fail") },
		},
		&config.PostConfig{},
		&stubImageAdapter{
			processFn: func(content string, userID uint) (string, error) { return content, nil },
			uploadFn:  func(data string, userID uint) (string, error) { return "", nil },
			deleteFn: func(path string) error {
				deleteImageCalls++
				return errors.New("img delete fail")
			},
			extractFn: func(content string) []string { return []string{"/a.png"} },
		},
		&stubTranslationAdapter{singleFn: func(text string) (string, error) { return "", nil }, markdownFn: func(content string) (string, error) { return "", nil }},
	)
	if err := svc.DeletePost(99, 1); err == nil {
		t.Fatalf("expected get fail")
	}
	if err := svc.DeletePost(1, 2); err == nil {
		t.Fatalf("expected unauthorized")
	}
	if err := svc.DeletePost(2, 1); err == nil {
		t.Fatalf("expected delete fail")
	}
	// cleanup errors should not fail the operation
	if err := svc.DeletePost(1, 1); err != nil {
		t.Fatalf("expected success despite cleanup errors: %v", err)
	}
	if deleteImageCalls == 0 {
		t.Fatalf("expected image delete attempts")
	}
}

func TestListTags(t *testing.T) {
	svc := newSvcForTest(
		&stubPostRepo{createFn: func(post *domain.Post) error { return nil }, getByID: func(id uint) (*domain.Post, error) { return nil, nil }, getAll: func(filter model.PostFilter) ([]*domain.Post, error) { return nil, nil }, updateFn: func(post *domain.Post) error { return nil }, deleteFn: func(id uint) error { return nil }},
		&stubTagRepo{
			attachFn:     func(postID uint, tagNames []string) error { return nil },
			replaceFn:    func(postID uint, tagNames []string) error { return nil },
			getTagsFn:    func(postID uint) ([]*domain.Tag, error) { return nil, nil },
			listTagsFn:   func() ([]*domain.Tag, error) { return []*domain.Tag{{ID: 1, Name: "go"}}, nil },
			deleteUnused: func(tagID uint) error { return nil },
		},
		&config.PostConfig{},
		&stubImageAdapter{processFn: func(content string, userID uint) (string, error) { return content, nil }, uploadFn: func(data string, userID uint) (string, error) { return "", nil }, deleteFn: func(path string) error { return nil }, extractFn: func(content string) []string { return nil }},
		&stubTranslationAdapter{singleFn: func(text string) (string, error) { return "", nil }, markdownFn: func(content string) (string, error) { return "", nil }},
	)
	tags, err := svc.ListTags()
	if err != nil || len(tags) != 1 {
		t.Fatalf("expected tags, got %v err=%v", tags, err)
	}
}

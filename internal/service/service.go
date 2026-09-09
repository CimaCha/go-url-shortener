package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	"sync"
)

const maxShortURLAttempts = 5

var (
	ErrURLHasGone     = errors.New("URL was deleted")
	ErrEmptyURL       = errors.New("empty URL")
	ErrEmptyURLList   = errors.New("empty URL list")
	ErrURLNotFound    = errors.New("URL not found")
	ErrUserNotFound   = errors.New("user not found")
	ErrRepository     = errors.New("error in repository")
	ErrFullURLExists  = errors.New("full URL already exists")
	ErrUniqueShortURL = errors.New("can't create unique short URL")
)

type Service struct {
	storage    URLStorage
	numWorkers int
}

type Result struct {
	shortURL    string
	userID      string
	originalURL string
	deletedFlag bool
	err         error
}

func NewService(storage URLStorage, numWorkers int) Service {
	return Service{
		storage:    storage,
		numWorkers: numWorkers,
	}
}

func (s Service) Shorten(ctx context.Context, fullURL string, userID string) (string, error) {
	if fullURL == "" {
		return "", ErrEmptyURL
	}
	for range maxShortURLAttempts {
		shortURL := rand.Text()
		storedShortURL, err := s.storage.SaveShortURL(ctx, shortURL, fullURL, userID)
		if err == nil {
			return shortURL, nil
		}
		if errors.Is(err, repository.ErrFullURLExists) {
			return storedShortURL, ErrFullURLExists
		}
		if !errors.Is(err, repository.ErrShortURLExists) {
			return "", fmt.Errorf("%w: save short URL: %w", ErrRepository, err)
		}
	}

	return "", ErrUniqueShortURL
}

func (s Service) Resolve(ctx context.Context, shortURL string) (string, error) {
	if shortURL == "" {
		return "", ErrEmptyURL
	}
	fullURL, err := s.storage.FindFullURL(ctx, shortURL)
	if err != nil {
		if errors.Is(err, repository.ErrURLNotFound) {
			return "", ErrURLNotFound
		}
		if errors.Is(err, repository.ErrURLHasGone) {
			return "", ErrURLHasGone
		}
		return "", ErrRepository
	}
	return fullURL, nil
}

func (s Service) ShortenBatch(ctx context.Context, fullURLBatch []*model.OriginalURLRecord, userID string) ([]*model.ShortURLRecord, error) {
	if len(fullURLBatch) == 0 {
		return nil, ErrEmptyURLList
	}

	for range maxShortURLAttempts {
		URLRecords := make([]*model.URLRecord, 0, len(fullURLBatch))
		shortURLRecords := make([]*model.ShortURLRecord, 0, len(fullURLBatch))
		for _, fullURL := range fullURLBatch {
			shortURL := rand.Text()
			URLRecords = append(URLRecords, &model.URLRecord{
				CorrelationID: fullURL.CorrelationID,
				OriginalURL:   fullURL.OriginalURL,
				ShortURL:      shortURL,
			})
			shortURLRecords = append(shortURLRecords, &model.ShortURLRecord{
				CorrelationID: fullURL.CorrelationID,
				ShortURL:      shortURL,
			})
		}

		err := s.storage.SaveShortURLBatch(ctx, URLRecords, userID)
		if err == nil {
			return shortURLRecords, nil
		}
		if !errors.Is(err, repository.ErrShortURLExists) {
			return nil, fmt.Errorf("%w: save short URL: %w", ErrRepository, err)
		}
	}

	return nil, ErrUniqueShortURL
}

func (s Service) GetUserURLs(ctx context.Context, userID string) ([]*model.UserRecord, error) {
	userURLsList, err := s.storage.GetUserURLs(ctx, userID)
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: get user URLs: %w", ErrRepository, err)
	}
	return userURLsList, nil
}

func (s Service) DeleteBatch(ctx context.Context, shortURLList []string, userID string) error {
	resultCh := make(chan Result)
	doneCh := make(chan struct{})
	var wg sync.WaitGroup
	resultArray := make([]string, 0)

	// получаем канал с данными из генератора
	inputChs := generator(shortURLList, doneCh)
	fanOutChs := s.fanOut(ctx, doneCh, inputChs, s.numWorkers)
	fanInCh := fanIn(doneCh, fanOutChs...)

	for res := range fanInCh {
		go func(res Result) {
			wg.Add(1)
			defer wg.Done()
			if res.userID == userID && !res.deletedFlag {
				resultCh <- res
			}
		}(res)
	}

	wg.Wait()

	for res := range resultCh {
		resultArray = append(resultArray, res.shortURL)
	}

	close(doneCh)

	err := s.storage.DeleteURLsBatch(ctx, resultArray)
	return err
}

func generator(shortURLsList []string, doneCh chan struct{}) chan string {
	inputCh := make(chan string)

	go func() {
		defer close(inputCh)

		for _, shortURL := range shortURLsList {
			select {
			case <-doneCh:
				return
			case inputCh <- shortURL:
			}
		}
	}()

	return inputCh
}

func (s Service) getShortURLData(ctx context.Context, doneCh chan struct{}, inputCh chan string) chan Result {
	res := make(chan Result)

	go func() {
		defer close(res)

		for data := range inputCh {
			// замедлим вычисление, как будто функция add требует больше вычислительных ресурсов
			shortURLData, err := s.storage.GetShortURLData(ctx, data)
			result := Result{
				shortURL:    data,
				userID:      shortURLData.UserID,
				originalURL: shortURLData.OriginalURL,
				deletedFlag: shortURLData.DeletedFlag,
				err:         err,
			}

			select {
			case <-doneCh:
				return
			case res <- result:
			}
		}
	}()
	return res
}

func (s Service) fanOut(ctx context.Context, doneCh chan struct{}, inputCh chan string, numWorkers int) []chan Result {
	channels := make([]chan Result, numWorkers)

	for i := 0; i < numWorkers; i++ {
		channels[i] = make(chan Result)
		// получаем канал из горутины add
		resultCh := s.getShortURLData(ctx, doneCh, inputCh)
		// отправляем его в слайс каналов
		channels[i] = resultCh
	}

	defer func() {
		// Закрываем все каналы воркеров
		for _, ch := range channels {
			close(ch)
		}
	}()

	return channels
}

func fanIn(doneCh chan struct{}, resultChs ...chan Result) chan Result {
	// конечный выходной канал в который отправляем данные из всех каналов из слайса, назовём его результирующим
	finalCh := make(chan Result)

	// понадобится для ожидания всех горутин
	var wg sync.WaitGroup

	// перебираем все входящие каналы
	for _, ch := range resultChs {
		// в горутину передавать переменную цикла нельзя, поэтому делаем так
		chClosure := ch

		// инкрементируем счётчик горутин, которые нужно подождать
		wg.Add(1)

		go func() {
			// откладываем сообщение о том, что горутина завершилась
			defer wg.Done()

			// получаем данные из канала
			for data := range chClosure {
				select {
				// выходим из горутины, если канал закрылся
				case <-doneCh:
					return
				// если не закрылся, отправляем данные в конечный выходной канал
				case finalCh <- data:
				}
			}
		}()
	}

	go func() {
		// ждём завершения всех горутин
		wg.Wait()
		// когда все горутины завершились, закрываем результирующий канал
		close(finalCh)
	}()

	// возвращаем результирующий канал
	return finalCh
}

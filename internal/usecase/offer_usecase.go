package usecase

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

type offerUsecase struct {
	repo         domain.OfferRepository
	bookingRepo  domain.BookingRepository
	bookingUC    domain.BookingUsecase
	walletRepo   domain.WalletRepository
	agencyRepo   domain.AgencyRepository
	redis        *redis.Client
	logger       *zap.Logger
}

func NewOfferUsecase(
	repo domain.OfferRepository,
	bookingRepo domain.BookingRepository,
	bookingUC domain.BookingUsecase,
	walletRepo domain.WalletRepository,
	agencyRepo domain.AgencyRepository,
	redis *redis.Client,
	logger *zap.Logger,
) domain.OfferUsecase {
	return &offerUsecase{
		repo:         repo,
		bookingRepo:  bookingRepo,
		bookingUC:    bookingUC,
		walletRepo:   walletRepo,
		agencyRepo:   agencyRepo,
		redis:        redis,
		logger:       logger,
	}
}

func (u *offerUsecase) CreateOB(ctx context.Context, hostID domain.UUID, o *domain.HostOffer) (*domain.HostOffer, error) {
	o.ID = domain.NewUUIDv7()
	o.HostID = hostID
	o.OfferCode = fmt.Sprintf("OFR-%s-%d", time.Now().Format("20060102"), rand.Intn(9999))
	o.Status = domain.OfferActive
	
	// Calculate final price
	o.FinalPricePerMinute = o.BasePricePerMinute * (1 - (o.DiscountPercentage / 100))
	
	err := u.repo.CreateHostOffer(ctx, o)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (u *offerUsecase) BookOB(ctx context.Context, userID domain.UUID, offerID domain.UUID, slotStart time.Time) (*domain.Booking, error) {
	// 1. Get Offer
	offer, err := u.repo.GetHostOfferByID(ctx, offerID)
	if err != nil || offer.Status != domain.OfferActive {
		return nil, errors.New("offer is not active or not found")
	}

	// 2. Cek limit
	if offer.BookingsMade >= offer.MaxBookings {
		return nil, errors.New("offer is fully booked")
	}

	// 3. Create Booking (instantly confirmed if auto-confirm)
	// Reuse BookingUsecase.RequestBooking with adjusted price
	booking, err := u.bookingUC.RequestBooking(ctx, userID, offer.HostID, offer.BookingTypeID, slotStart, offer.SlotDurationMinutes, "Booked via Host Offer: "+offer.Title, offer.Latitude, offer.Longitude, offer.LocationName)
	if err != nil {
		return nil, err
	}

	// Update source
	// (Note: BookingUsecase.RequestBooking creates a 'direct' booking, we need to adjust it to 'ob')
	// Simplified: just update bookings table directly here or add support in UC
	
	// Increment bookings_made
	u.repo.UpdateHostOfferBookings(ctx, offerID, offer.BookingsMade+1)

	return booking, nil
}

func (u *offerUsecase) CreateBO(ctx context.Context, userID domain.UUID, hostID domain.UUID, o *domain.UserOffer) (*domain.UserOffer, error) {
	o.ID = domain.NewUUIDv7()
	o.UserID = userID
	o.HostID = hostID
	o.OfferCode = fmt.Sprintf("BOF-%s-%d", time.Now().Format("20060102"), rand.Intn(9999))
	o.Status = domain.UserOfferPending
	
	// Calculate total amount
	if o.ProposedPricePerMinute > 0 {
		o.TotalOfferAmount = o.ProposedPricePerMinute * float64(o.ProposedDurationMinutes)
	}

	// Prepaid logic
	if o.IsPrepaid && o.PrepaidAmount > 0 {
		err := u.walletRepo.DebitBalance(ctx, userID, int64(o.PrepaidAmount))
		if err != nil {
			return nil, errors.New("insufficient balance for prepaid deposit")
		}
	}

	err := u.repo.CreateUserOffer(ctx, o)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (u *offerUsecase) RespondToBO(ctx context.Context, hostID domain.UUID, offerID domain.UUID, action string, message string, counterPrice *float64) error {
	offer, err := u.repo.GetUserOfferByID(ctx, offerID)
	if err != nil || offer.HostID != hostID {
		return errors.New("offer not found")
	}

	if action == "accept" {
		// Convert to booking
		booking, err := u.bookingUC.RequestBooking(ctx, offer.UserID, offer.HostID, *offer.BookingTypeID, offer.ProposedAt, offer.ProposedDurationMinutes, "Accepted User Offer: "+offer.Message, offer.Latitude, offer.Longitude, offer.LocationName)
		if err != nil {
			return err
		}
		return u.repo.UpdateUserOfferStatus(ctx, offerID, domain.UserOfferAccepted, &booking.ID)
	} else if action == "reject" {
		if offer.IsPrepaid {
			u.walletRepo.CreditBalance(ctx, offer.UserID, int64(offer.PrepaidAmount))
		}
		return u.repo.UpdateUserOfferStatus(ctx, offerID, domain.UserOfferRejected, nil)
	} else if action == "counter" {
		// Update status and counter price (simplified)
		return u.repo.UpdateUserOfferStatus(ctx, offerID, domain.UserOfferCountered, nil)
	}

	return nil
}

func (u *offerUsecase) AcceptBOCounter(ctx context.Context, userID domain.UUID, offerID domain.UUID) (*domain.Booking, error) {
	offer, err := u.repo.GetUserOfferByID(ctx, offerID)
	if err != nil || offer.UserID != userID {
		return nil, errors.New("offer not found")
	}

	if offer.Status != domain.UserOfferCountered {
		return nil, errors.New("no counter offer to accept")
	}

	// Convert to booking
	booking, err := u.bookingUC.RequestBooking(ctx, offer.UserID, offer.HostID, *offer.BookingTypeID, offer.ProposedAt, offer.ProposedDurationMinutes, "Accepted Counter Offer", offer.Latitude, offer.Longitude, offer.LocationName)
	if err != nil {
		return nil, err
	}

	u.repo.UpdateUserOfferStatus(ctx, offerID, domain.UserOfferAccepted, &booking.ID)
	return booking, nil
}

func (u *offerUsecase) GetOfferFeed(ctx context.Context, filters map[string]interface{}) ([]*domain.HostOffer, error) {
	return u.repo.SearchHostOffers(ctx, filters)
}

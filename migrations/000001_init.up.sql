create table public.auctions
(
    tender_id     uuid                                   not null
        primary key,
    start_price   bigint                                 not null,
    step          bigint                                 not null,
    start_at      timestamp with time zone               not null,
    end_at        timestamp with time zone               not null,
    created_by    uuid                                   not null,
    status        varchar(255)                           not null,
    created_at    timestamp with time zone default now() not null,
    current_price bigint                   default 0     not null,
    winner_id     uuid,
    updated_at    timestamp with time zone default now() not null,
    winner_bid_id bigint
);

alter table public.auctions
    owner to auction;

create table public.bids
(
    id          bigserial
        primary key,
    tender_id   uuid                                   not null
        references public.auctions,
    company_id  uuid                                   not null,
    bid_amount  bigint                                 not null,
    created_at  timestamp with time zone default now() not null,
    person_id   uuid
);

alter table public.bids
    owner to auction;

alter table public.auctions
    add constraint fk_auctions_winner_bid
        foreign key (winner_bid_id) references public.bids (id);

create index idx_bids_tender_id
    on public.bids (tender_id);

create index idx_bids_tender_time
    on public.bids (tender_id asc, created_at desc);

create index idx_bids_company
    on public.bids (company_id);

create index idx_bids_person
    on public.bids (person_id);

create table public.auction_participants
(
    tender_id  uuid                                   not null
        references public.auctions,
    company_id uuid                                   not null,
    created_at timestamp with time zone default now() not null,
    primary key (tender_id, company_id)
);

alter table public.auction_participants
    owner to auction;

create index idx_participants_company
    on public.auction_participants (company_id);


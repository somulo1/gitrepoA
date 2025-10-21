package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Initialize creates and returns a database connection
func Initialize(databaseURL string) (*sql.DB, error) {
	// Add SQLite-specific parameters for better concurrent access
	if databaseURL == "vaultke.db" {
		databaseURL = "vaultke.db?_busy_timeout=30000&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=1000&_foreign_keys=1"
	}

	db, err := sql.Open("sqlite3", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for better performance
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(0) // No limit

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set SQLite pragmas for better concurrent access
	pragmas := []string{
		"PRAGMA busy_timeout = 30000",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = 1000",
		"PRAGMA foreign_keys = ON",
		"PRAGMA temp_store = memory",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			log.Printf("Warning: failed to set pragma %s: %v", pragma, err)
		}
	}

	log.Println("Database connection established successfully")
	return db, nil
}

// Migrate runs database migrations
func Migrate(db *sql.DB) error {
	migrations := []string{
		createUsersTable,
		createChamasTable,
		createChamaMembersTable,
		createWalletsTable,
		createTransactionsTable,
		createNotificationsTable,
		createChatRoomsTable,
		createChatMessagesTable,
		createChatRoomMembersTable,
		createLoansTable,
		createGuarantorsTable,
		createLoanPaymentsTable,
		createMeetingsTable,
		createMeetingAttendanceTable,
		createMeetingDocumentsTable,
		createMeetingMinutesTable,
		createVotesTable,
		createVoteOptionsTable,
		createUserVotesTable,
		createMerryGoRoundTable,
		createMerryGoRoundParticipantsTable,
		createWelfareTable,
		createWelfareContributionsTable,
		createWelfareRequestsTable,
		addWelfareRequestBeneficiaryField,
		addUserProfileFields,
		addChatMessageFields,
		addWelfareContributionFields,
		addMeetingDocumentFileUrl,
		createChamaInvitationsTable,
		addChamaPermissionsColumn,
		addInvitationRoleColumns,
		createLearningTables,
		createRemindersTable,
		createSharesAndDividendsTables,
		// createPollsAndVotingTables, // DISABLED: Conflicts with existing vote system
		createDisbursementTables,
		addEnhancedLearningContentFields,
		createQuizResultsTable,
		addChamaCategoryColumn,
		addRecipientIDToTransactionsMigration,
		addDividendTypeColumnMigration,
		createDevicesTable,
		createSignalIdentityKeysTable,
		createSignalPreKeysTable,
		createSignalSignedPreKeysTable,
		createSignalSessionsTable,
		createSignalMessagesTable,
		createE2EEKeyBundlesTable,
		createE2EESessionsTable,
	}

	for i, migration := range migrations {
		// Handle special migrations that need custom logic
		if i == len(migrations)-10 { // Tenth to last migration is addWelfareRequestBeneficiaryField
			if err := addMissingWelfareRequestBeneficiaryField(db); err != nil {
				return fmt.Errorf("failed to add welfare request beneficiary field: %w", err)
			}
		} else if i == len(migrations)-9 { // Ninth to last migration is addUserProfileFields
			if err := addMissingUserProfileFields(db); err != nil {
				return fmt.Errorf("failed to add user profile fields: %w", err)
			}
		} else if i == len(migrations)-8 { // Eighth to last migration is addChatMessageFields
			if err := addMissingChatMessageFields(db); err != nil {
				return fmt.Errorf("failed to add chat message fields: %w", err)
			}
		} else if i == len(migrations)-7 { // Seventh to last migration is addWelfareContributionFields
			if err := addMissingWelfareContributionFields(db); err != nil {
				return fmt.Errorf("failed to add welfare contribution fields: %w", err)
			}
		} else if i == len(migrations)-6 { // Sixth to last migration is addMeetingDocumentFileUrl
			if err := addMissingMeetingDocumentFileUrl(db); err != nil {
				return fmt.Errorf("failed to add meeting document file_url field: %w", err)
			}
		} else if i == len(migrations)-5 { // Fifth to last migration is addChamaPermissionsColumn
			if err := addMissingChamaPermissionsColumn(db); err != nil {
				return fmt.Errorf("failed to add chama permissions column: %w", err)
			}
		} else if i == len(migrations)-4 { // Fourth to last migration is addInvitationRoleColumns
			if err := addMissingInvitationRoleColumns(db); err != nil {
				return fmt.Errorf("failed to add invitation role columns: %w", err)
			}
		} else if i == len(migrations)-3 { // Third to last migration is addEnhancedLearningContentFields
			if err := addMissingEnhancedLearningContentFields(db); err != nil {
				return fmt.Errorf("failed to add enhanced learning content fields: %w", err)
			}
		} else if i == len(migrations)-2 { // Second to last migration is createQuizResultsTable
			if _, err := db.Exec(migration); err != nil {
				return fmt.Errorf("failed to create quiz results table: %w", err)
			}
		} else if i == len(migrations)-1 { // Last migration is addDividendTypeColumn
			if err := addDividendTypeColumn(db); err != nil {
				return fmt.Errorf("failed to add dividend_type column to dividend_declarations table: %w", err)
			}
		} else if i == len(migrations)-2 { // Second to last migration is addRecipientIDToTransactionsMigration
			if err := addRecipientIDToTransactions(db); err != nil {
				return fmt.Errorf("failed to add recipient_id to transactions table: %w", err)
			}
		} else if i == len(migrations)-2 { // Third to last migration is addChamaCategoryColumn
			if err := addCategoryColumnToChamasTable(db); err != nil {
				return fmt.Errorf("failed to add chama category column: %w", err)
			}
		} else if i == len(migrations) { // Last migration is chat_message_refactor
			if err := updateChatMessageStorage(db); err != nil {
				return fmt.Errorf("failed to update chat message storage: %w", err)
			}
		} else if i == len(migrations)+1 { // New migration for chat message content refactor
			if err := refactorChatMessageContent(db); err != nil {
				return fmt.Errorf("failed to refactor chat message content: %w", err)
			}
		} else {
			// Regular migrations
			if _, err := db.Exec(migration); err != nil {
				return fmt.Errorf("failed to run migration %d: %w", i+1, err)
			}
		}
	}

	log.Println("Database migrations completed successfully")
	return nil
}

const addRecipientIDToTransactionsMigration = "SELECT 1"

const addDividendTypeColumnMigration = "SELECT 1" // Placeholder for migration function

// addDividendTypeColumn adds dividend_type column to dividend_declarations table
func addRecipientIDToTransactions(db *sql.DB) error {
	// Check if recipient_id column exists
	var exists bool
	query := `SELECT COUNT(*) FROM pragma_table_info('transactions') WHERE name = 'recipient_id'`
	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if recipient_id column exists: %w", err)
	}

	// Add column if it doesn't exist
	if !exists {
		alterQuery := `ALTER TABLE transactions ADD COLUMN recipient_id TEXT`
		if _, err := db.Exec(alterQuery); err != nil {
			return fmt.Errorf("failed to add recipient_id column: %w", err)
		}
		log.Printf("Added recipient_id column to transactions table")

		// Create index for recipient_id
		indexQuery := `CREATE INDEX IF NOT EXISTS idx_transactions_recipient_id ON transactions(recipient_id)`
		if _, err := db.Exec(indexQuery); err != nil {
			log.Printf("Warning: failed to create index for recipient_id: %v", err)
		} else {
			log.Printf("Created index for recipient_id on transactions table")
		}
	} else {
		log.Printf("Column recipient_id already exists in transactions table")
	}

	return nil
}

// addDividendTypeColumn adds dividend_type column to dividend_declarations table
func addDividendTypeColumn(db *sql.DB) error {
	// Check if total_dividend_amount column exists and rename to total_amount
	var exists bool
	query := `SELECT COUNT(*) FROM pragma_table_info('dividend_declarations') WHERE name = 'total_dividend_amount'`
	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if total_dividend_amount column exists: %w", err)
	}

	if exists {
		// Rename column
		renameQuery := `ALTER TABLE dividend_declarations RENAME COLUMN total_dividend_amount TO total_amount`
		if _, err := db.Exec(renameQuery); err != nil {
			return fmt.Errorf("failed to rename total_dividend_amount to total_amount: %w", err)
		}
		log.Printf("Renamed total_dividend_amount to total_amount in dividend_declarations table")
	}

	// Check if dividend_type column exists
	query = `SELECT COUNT(*) FROM pragma_table_info('dividend_declarations') WHERE name = 'dividend_type'`
	err = db.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if dividend_type column exists: %w", err)
	}

	// Add column if it doesn't exist
	if !exists {
		alterQuery := `ALTER TABLE dividend_declarations ADD COLUMN dividend_type TEXT DEFAULT 'cash'`
		if _, err := db.Exec(alterQuery); err != nil {
			return fmt.Errorf("failed to add dividend_type column: %w", err)
		}
		log.Printf("Added dividend_type column to dividend_declarations table")
	} else {
		log.Printf("Column dividend_type already exists in dividend_declarations table")
	}

	// Add missing columns from the updated schema
	columnsToAdd := []struct {
		name         string
		dataType     string
		defaultValue string
	}{
		{"declaration_date", "DATETIME", "CURRENT_TIMESTAMP"},
		{"eligibility_criteria", "TEXT", "NULL"},
		{"approval_required", "BOOLEAN", "TRUE"},
		{"created_by", "TEXT", "NULL"},
		{"created_by_id", "TEXT", "NULL"},
		{"timestamp", "DATETIME", "NULL"},
		{"transaction_id", "TEXT", "NULL"},
		{"security_hash", "TEXT", "NULL"},
	}

	for _, col := range columnsToAdd {
		var exists bool
		query = `SELECT COUNT(*) FROM pragma_table_info('dividend_declarations') WHERE name = ?`
		err = db.QueryRow(query, col.name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if %s column exists: %w", col.name, err)
		}

		if !exists {
			alterQuery := fmt.Sprintf("ALTER TABLE dividend_declarations ADD COLUMN %s %s DEFAULT %s", col.name, col.dataType, col.defaultValue)
			if _, err := db.Exec(alterQuery); err != nil {
				return fmt.Errorf("failed to add %s column: %w", col.name, err)
			}
			log.Printf("Added %s column to dividend_declarations table", col.name)
		}
	}

	return nil
}

// SQL migration statements
const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    phone TEXT UNIQUE NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    avatar TEXT,
    role TEXT NOT NULL DEFAULT 'user',
    status TEXT NOT NULL DEFAULT 'pending',
    is_email_verified BOOLEAN DEFAULT FALSE,
    is_phone_verified BOOLEAN DEFAULT FALSE,
    language TEXT DEFAULT 'en',
    theme TEXT DEFAULT 'dark',
    county TEXT,
    town TEXT,
    latitude REAL,
    longitude REAL,
    business_type TEXT,
    business_description TEXT,
    rating REAL DEFAULT 0,
    total_ratings INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

const createDevicesTable = `
CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    device_id INTEGER NOT NULL,
    device_name TEXT,
    device_type TEXT, -- 'mobile', 'desktop', 'web'
    registration_id INTEGER,
    signed_pre_key_id INTEGER,
    last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, device_id)
);`

const createSignalIdentityKeysTable = `
CREATE TABLE IF NOT EXISTS signal_identity_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    device_id INTEGER NOT NULL,
    public_key TEXT NOT NULL,
    private_key TEXT NOT NULL, -- Encrypted
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id, device_id) REFERENCES devices(user_id, device_id) ON DELETE CASCADE,
    UNIQUE(user_id, device_id)
);`

const createSignalPreKeysTable = `
CREATE TABLE IF NOT EXISTS signal_pre_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    device_id INTEGER NOT NULL,
    pre_key_id INTEGER NOT NULL,
    public_key TEXT NOT NULL,
    private_key TEXT NOT NULL, -- Encrypted
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id, device_id) REFERENCES devices(user_id, device_id) ON DELETE CASCADE,
    UNIQUE(user_id, device_id, pre_key_id)
);`

const createSignalSignedPreKeysTable = `
CREATE TABLE IF NOT EXISTS signal_signed_pre_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    device_id INTEGER NOT NULL,
    signed_pre_key_id INTEGER NOT NULL,
    public_key TEXT NOT NULL,
    private_key TEXT NOT NULL, -- Encrypted
    signature TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id, device_id) REFERENCES devices(user_id, device_id) ON DELETE CASCADE,
    UNIQUE(user_id, device_id, signed_pre_key_id)
);`

const createSignalSessionsTable = `
CREATE TABLE IF NOT EXISTS signal_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    device_id INTEGER NOT NULL,
    session_id TEXT NOT NULL, -- Base64 encoded session identifier
    session_data TEXT NOT NULL, -- Encrypted session data
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id, device_id) REFERENCES devices(user_id, device_id) ON DELETE CASCADE,
    UNIQUE(user_id, device_id, session_id)
);`

const createSignalMessagesTable = `
CREATE TABLE IF NOT EXISTS signal_messages (
    id TEXT PRIMARY KEY,
    sender_id TEXT NOT NULL,
    sender_device_id INTEGER NOT NULL,
    recipient_id TEXT NOT NULL,
    recipient_device_id INTEGER NOT NULL,
    message_type TEXT NOT NULL, -- 'message', 'pre_key_bundle'
    ciphertext TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_delivered BOOLEAN DEFAULT FALSE,
    is_read BOOLEAN DEFAULT FALSE,
    metadata TEXT, -- JSON metadata
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sender_id, sender_device_id) REFERENCES devices(user_id, device_id) ON DELETE CASCADE,
    FOREIGN KEY (recipient_id, recipient_device_id) REFERENCES devices(user_id, device_id) ON DELETE CASCADE
);`

const createE2EEKeyBundlesTable = `
CREATE TABLE IF NOT EXISTS e2ee_key_bundles (
    user_id TEXT PRIMARY KEY,
    identity_key TEXT NOT NULL,
    signed_pre_key TEXT NOT NULL,
    pre_key_signature TEXT NOT NULL,
    one_time_pre_keys TEXT NOT NULL, -- JSON array
    registration_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

const createE2EESessionsTable = `
CREATE TABLE IF NOT EXISTS e2ee_sessions (
    id TEXT PRIMARY KEY,
    user_a_id TEXT NOT NULL,
    user_b_id TEXT NOT NULL,
    shared_secret TEXT NOT NULL,
    sending_chain TEXT NOT NULL,
    receiving_chain TEXT NOT NULL,
    message_number INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_a_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (user_b_id) REFERENCES users(id) ON DELETE CASCADE
);`

const createChamasTable = `
CREATE TABLE IF NOT EXISTS chamas (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    category TEXT NOT NULL DEFAULT 'chama', -- 'chama' or 'contribution'
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    avatar TEXT,
    county TEXT NOT NULL,
    town TEXT NOT NULL,
    latitude REAL,
    longitude REAL,
    contribution_amount REAL NOT NULL,
    contribution_frequency TEXT NOT NULL,
    target_amount REAL, -- For contribution groups
    target_deadline DATETIME, -- For contribution groups
    payment_method TEXT, -- 'till' or 'paybill'
    till_number TEXT, -- For TILL payments
    paybill_business_number TEXT, -- For PAYBILL payments
    paybill_account_number TEXT, -- For PAYBILL payments
    payment_recipient_name TEXT, -- Name user should expect on successful payment
    max_members INTEGER,
    current_members INTEGER DEFAULT 0,
    total_funds REAL DEFAULT 0,
    is_public BOOLEAN DEFAULT FALSE,
    requires_approval BOOLEAN DEFAULT TRUE,
    rules TEXT, -- JSON array of rules
    meeting_frequency TEXT,
    meeting_day_of_week INTEGER,
    meeting_day_of_month INTEGER,
    meeting_time TEXT,
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by) REFERENCES users(id)
);`

const createChamaMembersTable = `
CREATE TABLE IF NOT EXISTS chama_members (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    total_contributions REAL DEFAULT 0,
    last_contribution DATETIME,
    rating REAL DEFAULT 0,
    total_ratings INTEGER DEFAULT 0,
    FOREIGN KEY (chama_id) REFERENCES chamas(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(chama_id, user_id)
);`

const createWalletsTable = `
CREATE TABLE IF NOT EXISTS wallets (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    balance REAL DEFAULT 0,
    currency TEXT DEFAULT 'KES',
    is_active BOOLEAN DEFAULT TRUE,
    is_locked BOOLEAN DEFAULT FALSE,
    daily_limit REAL,
    monthly_limit REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

const createTransactionsTable = `
CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    from_wallet_id TEXT,
    to_wallet_id TEXT,
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    amount REAL NOT NULL,
    currency TEXT DEFAULT 'KES',
    description TEXT,
    reference TEXT,
    payment_method TEXT NOT NULL,
    metadata TEXT, -- JSON metadata
    fees REAL DEFAULT 0,
    initiated_by TEXT NOT NULL,
    approved_by TEXT,
    requires_approval BOOLEAN DEFAULT FALSE,
    approval_deadline DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_wallet_id) REFERENCES wallets(id),
    FOREIGN KEY (to_wallet_id) REFERENCES wallets(id),
    FOREIGN KEY (initiated_by) REFERENCES users(id),
    FOREIGN KEY (approved_by) REFERENCES users(id)
);`

const createNotificationsTable = `
CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    type TEXT NOT NULL,
    priority TEXT DEFAULT 'normal',
    category TEXT NULL,
    reference_type TEXT NULL,
    reference_id INTEGER NULL,
    status TEXT DEFAULT 'pending',
    is_read BOOLEAN DEFAULT FALSE,
    read_at DATETIME NULL,
    scheduled_for DATETIME DEFAULT CURRENT_TIMESTAMP,
    sent_at DATETIME NULL,
    delivered_at DATETIME NULL,
    data TEXT, -- JSON data
    sound_played BOOLEAN DEFAULT FALSE,
    retry_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);`

const createChatRoomsTable = `
CREATE TABLE IF NOT EXISTS chat_rooms (
    id TEXT PRIMARY KEY,
    name TEXT,
    type TEXT NOT NULL, -- 'private', 'group', 'chama'
    chama_id TEXT,
    created_by TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    last_message TEXT,
    last_message_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);`

const createChatMessagesTable = `
CREATE TABLE IF NOT EXISTS chat_messages (
    id TEXT PRIMARY KEY,
    room_id TEXT NOT NULL,
    sender_id TEXT NOT NULL,
    message TEXT NOT NULL,
    content TEXT NOT NULL,
    type TEXT DEFAULT 'text',
    message_type TEXT DEFAULT 'text', -- 'text', 'image', 'file', 'voice'
    metadata TEXT DEFAULT '{}',
    file_url TEXT,
    is_edited BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,
    reply_to TEXT,
    reply_to_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES chat_rooms(id),
    FOREIGN KEY (sender_id) REFERENCES users(id),
    FOREIGN KEY (reply_to) REFERENCES chat_messages(id),
    FOREIGN KEY (reply_to_id) REFERENCES chat_messages(id)
);`

const createMeetingsTable = `
CREATE TABLE IF NOT EXISTS meetings (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    scheduled_at DATETIME NOT NULL,
    duration INTEGER, -- in minutes
    location TEXT,
    meeting_url TEXT,
    meeting_type TEXT NOT NULL DEFAULT 'physical', -- 'physical', 'virtual', 'hybrid'
    livekit_room_name TEXT, -- LiveKit room identifier
    livekit_room_id TEXT, -- LiveKit room ID
    status TEXT NOT NULL DEFAULT 'scheduled', -- 'scheduled', 'active', 'ended', 'cancelled'
    started_at DATETIME,
    ended_at DATETIME,
    recording_enabled BOOLEAN DEFAULT FALSE,
    recording_url TEXT,
    transcript_url TEXT,
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);`

const createMeetingAttendanceTable = `
CREATE TABLE IF NOT EXISTS meeting_attendance (
    id TEXT PRIMARY KEY,
    meeting_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    attendance_type TEXT NOT NULL, -- 'physical', 'virtual'
    joined_at DATETIME,
    left_at DATETIME,
    duration_minutes INTEGER DEFAULT 0,
    is_present BOOLEAN DEFAULT FALSE,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (meeting_id) REFERENCES meetings(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(meeting_id, user_id)
);`

const createMeetingDocumentsTable = `
CREATE TABLE IF NOT EXISTS meeting_documents (
    id TEXT PRIMARY KEY,
    meeting_id TEXT NOT NULL,
    uploaded_by TEXT NOT NULL,
    file_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_url TEXT NOT NULL,
    file_size INTEGER,
    file_type TEXT,
    document_type TEXT, -- 'agenda', 'minutes', 'attachment', 'recording'
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (meeting_id) REFERENCES meetings(id),
    FOREIGN KEY (uploaded_by) REFERENCES users(id)
);`

const createMeetingMinutesTable = `
CREATE TABLE IF NOT EXISTS meeting_minutes (
    id TEXT PRIMARY KEY,
    meeting_id TEXT NOT NULL,
    content TEXT NOT NULL,
    taken_by TEXT NOT NULL, -- Secretary or authorized user
    status TEXT NOT NULL DEFAULT 'draft', -- 'draft', 'approved', 'published'
    approved_by TEXT,
    approved_at DATETIME,
    version INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (meeting_id) REFERENCES meetings(id),
    FOREIGN KEY (taken_by) REFERENCES users(id),
    FOREIGN KEY (approved_by) REFERENCES users(id)
);`

const createVotesTable = `
CREATE TABLE IF NOT EXISTS votes (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL DEFAULT 'single', -- 'single', 'multiple'
    status TEXT NOT NULL DEFAULT 'active',
    starts_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    ends_at DATETIME NOT NULL,
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);`

const createVoteOptionsTable = `
CREATE TABLE IF NOT EXISTS vote_options (
    id TEXT PRIMARY KEY,
    vote_id TEXT NOT NULL,
    option_text TEXT NOT NULL,
    vote_count INTEGER DEFAULT 0,
    FOREIGN KEY (vote_id) REFERENCES votes(id)
);`

const createUserVotesTable = `
CREATE TABLE IF NOT EXISTS user_votes (
    id TEXT PRIMARY KEY,
    vote_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    option_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (vote_id) REFERENCES votes(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (option_id) REFERENCES vote_options(id),
    UNIQUE(vote_id, user_id, option_id)
);`

const createChatRoomMembersTable = `
CREATE TABLE IF NOT EXISTS chat_room_members (
    id TEXT PRIMARY KEY,
    room_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_read_at DATETIME,
    is_active BOOLEAN DEFAULT TRUE,
    is_muted BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (room_id) REFERENCES chat_rooms(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(room_id, user_id)
);`

const createLoansTable = `
CREATE TABLE IF NOT EXISTS loans (
    id TEXT PRIMARY KEY,
    borrower_id TEXT NOT NULL,
    chama_id TEXT NOT NULL,
    type TEXT NOT NULL,
    amount REAL NOT NULL,
    interest_rate REAL DEFAULT 0,
    duration INTEGER NOT NULL,
    purpose TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    approved_by TEXT,
    approved_at DATETIME,
    disbursed_at DATETIME,
    due_date DATETIME,
    total_amount REAL DEFAULT 0,
    paid_amount REAL DEFAULT 0,
    remaining_amount REAL DEFAULT 0,
    required_guarantors INTEGER NOT NULL,
    approved_guarantors INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (borrower_id) REFERENCES users(id),
    FOREIGN KEY (chama_id) REFERENCES chamas(id),
    FOREIGN KEY (approved_by) REFERENCES users(id)
);`

const createGuarantorsTable = `
CREATE TABLE IF NOT EXISTS guarantors (
    id TEXT PRIMARY KEY,
    loan_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    amount REAL NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    message TEXT,
    responded_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (loan_id) REFERENCES loans(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(loan_id, user_id)
);`

const createLoanPaymentsTable = `
CREATE TABLE IF NOT EXISTS loan_payments (
    id TEXT PRIMARY KEY,
    loan_id TEXT NOT NULL,
    amount REAL NOT NULL,
    principal_amount REAL NOT NULL,
    interest_amount REAL NOT NULL,
    payment_method TEXT NOT NULL,
    reference TEXT,
    paid_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (loan_id) REFERENCES loans(id)
);`

const createMerryGoRoundTable = `
CREATE TABLE IF NOT EXISTS merry_go_rounds (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    amount_per_round REAL NOT NULL,
    frequency TEXT NOT NULL, -- 'weekly', 'monthly'
    total_participants INTEGER NOT NULL,
    current_round INTEGER DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'active',
    start_date DATE NOT NULL,
    next_payout_date DATE,
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);`

const createMerryGoRoundParticipantsTable = `
CREATE TABLE IF NOT EXISTS merry_go_round_participants (
    id TEXT PRIMARY KEY,
    merry_go_round_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    has_received BOOLEAN DEFAULT FALSE,
    received_at DATETIME,
    total_contributed REAL DEFAULT 0,
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (merry_go_round_id) REFERENCES merry_go_rounds(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(merry_go_round_id, user_id),
    UNIQUE(merry_go_round_id, position)
);`

const createWelfareTable = `
CREATE TABLE IF NOT EXISTS welfare_funds (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    target_amount REAL,
    current_amount REAL DEFAULT 0,
    contribution_per_member REAL,
    purpose TEXT NOT NULL, -- 'emergency', 'medical', 'funeral', 'education'
    status TEXT NOT NULL DEFAULT 'active',
    beneficiary_id TEXT,
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id),
    FOREIGN KEY (beneficiary_id) REFERENCES users(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);`

const createWelfareContributionsTable = `
CREATE TABLE IF NOT EXISTS welfare_contributions (
    id TEXT PRIMARY KEY,
    welfare_fund_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    amount REAL NOT NULL,
    payment_method TEXT NOT NULL,
    reference TEXT,
    contributed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (welfare_fund_id) REFERENCES welfare_funds(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);`

const createWelfareRequestsTable = `
CREATE TABLE IF NOT EXISTS welfare_requests (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    requester_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    amount REAL NOT NULL,
    category TEXT NOT NULL,
    urgency TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    votes_for INTEGER DEFAULT 0,
    votes_against INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id),
    FOREIGN KEY (requester_id) REFERENCES users(id)
);`

// addWelfareRequestBeneficiaryField adds beneficiary_id field to welfare_requests table
const addWelfareRequestBeneficiaryField = "SELECT 1" // Placeholder - actual logic in migration function

// addUserProfileFields adds missing profile fields if they don't exist
const addUserProfileFields = "SELECT 1" // Placeholder - actual logic in migration function

// addChatMessageFields adds missing chat message fields if they don't exist
const addChatMessageFields = "SELECT 1" // Placeholder - actual logic in migration function

// addChatRoomFields adds missing chat room fields if they don't exist
const addChatRoomFields = "SELECT 1" // Placeholder - actual logic in migration function

// addMissingUserProfileFields safely adds missing profile fields to users table
func addMissingUserProfileFields(db *sql.DB) error {
	// Check if columns exist and add them if they don't
	columns := []struct {
		name     string
		dataType string
	}{
		{"bio", "TEXT"},
		{"occupation", "TEXT"},
		{"date_of_birth", "DATE"},
		{"gender", "TEXT"},
	}

	for _, col := range columns {
		// Check if column exists
		var exists bool
		query := `SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = ?`
		err := db.QueryRow(query, col.name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if column %s exists: %w", col.name, err)
		}

		// Add column if it doesn't exist
		if !exists {
			alterQuery := fmt.Sprintf("ALTER TABLE users ADD COLUMN %s %s", col.name, col.dataType)
			if _, err := db.Exec(alterQuery); err != nil {
				return fmt.Errorf("failed to add column %s: %w", col.name, err)
			}
			log.Printf("Added column %s to users table", col.name)
		} else {
			log.Printf("Column %s already exists in users table", col.name)
		}
	}

	return nil
}

// addMissingWelfareRequestBeneficiaryField safely adds beneficiary_id field to welfare_requests table
func addMissingWelfareRequestBeneficiaryField(db *sql.DB) error {
	// Check if beneficiary_id column exists
	var exists bool
	query := `SELECT COUNT(*) FROM pragma_table_info('welfare_requests') WHERE name = 'beneficiary_id'`
	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if beneficiary_id column exists: %w", err)
	}

	// Add column if it doesn't exist
	if !exists {
		alterQuery := `ALTER TABLE welfare_requests ADD COLUMN beneficiary_id TEXT`
		if _, err := db.Exec(alterQuery); err != nil {
			return fmt.Errorf("failed to add beneficiary_id column: %w", err)
		}
		log.Printf("Added beneficiary_id column to welfare_requests table")

		// Set beneficiary_id to requester_id for existing records (self-requests)
		updateQuery := `UPDATE welfare_requests SET beneficiary_id = requester_id WHERE beneficiary_id IS NULL`
		if _, err := db.Exec(updateQuery); err != nil {
			log.Printf("Warning: failed to update existing records with beneficiary_id: %v", err)
		} else {
			log.Printf("Updated existing welfare requests to set beneficiary_id = requester_id")
		}
	} else {
		log.Printf("Column beneficiary_id already exists in welfare_requests table")
	}

	return nil
}

// addMissingChatMessageFields safely adds missing chat message fields to chat_messages table
func addMissingChatMessageFields(db *sql.DB) error {
	// Check if columns exist and add them if they don't
	columns := []struct {
		name         string
		dataType     string
		defaultValue string
	}{
		{"content", "TEXT", "''"},
		{"type", "TEXT", "'text'"},
		{"metadata", "TEXT", "'{}'"},
		{"is_deleted", "BOOLEAN", "FALSE"},
		{"reply_to_id", "TEXT", "NULL"},
	}

	for _, col := range columns {
		// Check if column exists
		var exists bool
		query := `SELECT COUNT(*) FROM pragma_table_info('chat_messages') WHERE name = ?`
		err := db.QueryRow(query, col.name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if column %s exists: %w", col.name, err)
		}

		// Add column if it doesn't exist
		if !exists {
			alterQuery := fmt.Sprintf("ALTER TABLE chat_messages ADD COLUMN %s %s DEFAULT %s", col.name, col.dataType, col.defaultValue)
			if _, err := db.Exec(alterQuery); err != nil {
				return fmt.Errorf("failed to add column %s: %w", col.name, err)
			}
			log.Printf("Added column %s to chat_messages table", col.name)
		} else {
			log.Printf("Column %s already exists in chat_messages table", col.name)
		}
	}

	// Update existing records to populate content from message if content is empty
	updateQuery := `UPDATE chat_messages SET content = message WHERE content = '' OR content IS NULL`
	if _, err := db.Exec(updateQuery); err != nil {
		log.Printf("Warning: failed to update content from message: %v", err)
	} else {
		log.Printf("Updated content field from message field for existing records")
	}

	// Update new records to populate message from content to maintain compatibility
	updateQuery2 := `UPDATE chat_messages SET message = content WHERE message = '' OR message IS NULL`
	if _, err := db.Exec(updateQuery2); err != nil {
		log.Printf("Warning: failed to update message from content: %v", err)
	} else {
		log.Printf("Updated message field from content field for compatibility")
	}

	return nil
}

// addMissingChatRoomFields safely adds missing chat room fields to chat_rooms table
func addMissingChatRoomFields(db *sql.DB) error {
	// Check if columns exist and add them if they don't
	columns := []struct {
		name         string
		dataType     string
		defaultValue string
	}{
		{"is_active", "BOOLEAN", "TRUE"},
		{"last_message", "TEXT", "NULL"},
		{"last_message_at", "DATETIME", "NULL"},
		{"updated_at", "DATETIME", "CURRENT_TIMESTAMP"},
	}

	for _, col := range columns {
		// Check if column exists
		var exists bool
		query := `SELECT COUNT(*) FROM pragma_table_info('chat_rooms') WHERE name = ?`
		err := db.QueryRow(query, col.name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if column %s exists: %w", col.name, err)
		}

		// Add column if it doesn't exist
		if !exists {
			alterQuery := fmt.Sprintf("ALTER TABLE chat_rooms ADD COLUMN %s %s DEFAULT %s", col.name, col.dataType, col.defaultValue)
			if _, err := db.Exec(alterQuery); err != nil {
				return fmt.Errorf("failed to add column %s: %w", col.name, err)
			}
			log.Printf("Added column %s to chat_rooms table", col.name)
		} else {
			log.Printf("Column %s already exists in chat_rooms table", col.name)
		}
	}

	return nil
}

// addMissingTransactionFields adds missing columns to the transactions table
func addMissingTransactionFields(db *sql.DB) error {
	columns := []struct {
		name         string
		dataType     string
		defaultValue string
	}{
		{"checkout_request_id", "TEXT", "NULL"},
	}

	for _, col := range columns {
		// Check if column exists
		var exists bool
		query := `SELECT COUNT(*) FROM pragma_table_info('transactions') WHERE name = ?`
		err := db.QueryRow(query, col.name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if column %s exists: %w", col.name, err)
		}

		// Add column if it doesn't exist
		if !exists {
			alterQuery := fmt.Sprintf("ALTER TABLE transactions ADD COLUMN %s %s DEFAULT %s", col.name, col.dataType, col.defaultValue)
			if _, err := db.Exec(alterQuery); err != nil {
				return fmt.Errorf("failed to add column %s: %w", col.name, err)
			}
			log.Printf("Added column %s to transactions table", col.name)
		} else {
			log.Printf("Column %s already exists in transactions table", col.name)
		}
	}

	return nil
}

// addTransactionFields is a placeholder for the migration system
const addTransactionFields = ""

// addWelfareContributionFields is a placeholder for the migration system
const addWelfareContributionFields = ""

// addMissingWelfareContributionFields adds missing columns to the welfare_contributions table
func addMissingWelfareContributionFields(db *sql.DB) error {
	columns := []struct {
		name         string
		dataType     string
		defaultValue string
	}{
		{"welfare_request_id", "TEXT", "NULL"},
		{"contributor_id", "TEXT", "NULL"},
		{"message", "TEXT", "NULL"},
		{"status", "TEXT", "'completed'"},
		{"created_at", "DATETIME", "CURRENT_TIMESTAMP"},
		{"updated_at", "DATETIME", "CURRENT_TIMESTAMP"},
	}

	for _, col := range columns {
		// Check if column exists
		var exists bool
		query := `SELECT COUNT(*) FROM pragma_table_info('welfare_contributions') WHERE name = ?`
		err := db.QueryRow(query, col.name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if column %s exists: %w", col.name, err)
		}

		// Add column if it doesn't exist
		if !exists {
			alterQuery := fmt.Sprintf("ALTER TABLE welfare_contributions ADD COLUMN %s %s DEFAULT %s", col.name, col.dataType, col.defaultValue)
			if _, err := db.Exec(alterQuery); err != nil {
				return fmt.Errorf("failed to add column %s: %w", col.name, err)
			}
			log.Printf("Added column %s to welfare_contributions table", col.name)
		} else {
			log.Printf("Column %s already exists in welfare_contributions table", col.name)
		}
	}

	return nil
}

// Migration constant for adding file_url to meeting_documents
const addMeetingDocumentFileUrl = `-- This is handled by addMissingMeetingDocumentFileUrl function`

// addMissingMeetingDocumentFileUrl adds the file_url column to meeting_documents table if it doesn't exist
func addMissingMeetingDocumentFileUrl(db *sql.DB) error {
	// Check if file_url column exists
	rows, err := db.Query("PRAGMA table_info(meeting_documents)")
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}
	defer rows.Close()

	hasFileUrl := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}

		if name == "file_url" {
			hasFileUrl = true
			break
		}
	}

	if !hasFileUrl {
		log.Println("Adding file_url column to meeting_documents table...")
		_, err = db.Exec("ALTER TABLE meeting_documents ADD COLUMN file_url TEXT")
		if err != nil {
			return fmt.Errorf("failed to add file_url column: %w", err)
		}
		log.Println("Successfully added file_url column to meeting_documents table")
	} else {
		log.Println("file_url column already exists in meeting_documents table")
	}

	return nil
}

// createChamaInvitationsTable creates the chama invitations table
const createChamaInvitationsTable = `
CREATE TABLE IF NOT EXISTS chama_invitations (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    inviter_id TEXT NOT NULL,
    email TEXT NOT NULL,
    phone_number TEXT,
    message TEXT,
    invitation_token TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected', 'expired')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    responded_at DATETIME,
    responded_by TEXT,
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (responded_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_chama_invitations_email ON chama_invitations(email);
CREATE INDEX IF NOT EXISTS idx_chama_invitations_chama_id ON chama_invitations(chama_id);
CREATE INDEX IF NOT EXISTS idx_chama_invitations_status ON chama_invitations(status);
CREATE INDEX IF NOT EXISTS idx_chama_invitations_token ON chama_invitations(invitation_token);
`

// addChamaPermissionsColumn adds permissions column to chamas table
const addChamaPermissionsColumn = `
-- This will be handled by addMissingChamaPermissionsColumn function
SELECT 1;
`

// addInvitationRoleColumns adds role-related columns to chama_invitations table
const addInvitationRoleColumns = `
-- This will be handled by addMissingInvitationRoleColumns function
SELECT 1;
`

// addMissingChamaPermissionsColumn adds permissions column to chamas table if it doesn't exist
func addMissingChamaPermissionsColumn(db *sql.DB) error {
	// Check if permissions column exists
	var columnExists bool
	checkQuery := `
		SELECT COUNT(*) > 0
		FROM pragma_table_info('chamas')
		WHERE name = 'permissions'
	`

	err := db.QueryRow(checkQuery).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check if permissions column exists: %w", err)
	}

	if !columnExists {
		log.Println("Adding permissions column to chamas table")
		addColumnQuery := `
			ALTER TABLE chamas ADD COLUMN permissions TEXT DEFAULT '{"allowMerryGoRound": true, "allowWelfare": true, "allowMarketplace": true}'
		`
		_, err = db.Exec(addColumnQuery)
		if err != nil {
			return fmt.Errorf("failed to add permissions column: %w", err)
		}
		log.Println("Successfully added permissions column to chamas table")
	} else {
		log.Println("Column permissions already exists in chamas table")
	}

	return nil
}

// addCategoryColumnToChamasTable adds the category column to the chamas table if it doesn't exist
func addCategoryColumnToChamasTable(db *sql.DB) error {
	// Check if category column exists
	var columnExists bool
	checkColumnQuery := `
		SELECT COUNT(*) > 0
		FROM pragma_table_info('chamas')
		WHERE name = 'category'
	`
	err := db.QueryRow(checkColumnQuery).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check if category column exists: %w", err)
	}

	if !columnExists {
		log.Println("Adding category column to chamas table")
		addColumnQuery := `
			ALTER TABLE chamas ADD COLUMN category TEXT NOT NULL DEFAULT 'chama'
		`
		_, err = db.Exec(addColumnQuery)
		if err != nil {
			return fmt.Errorf("failed to add category column: %w", err)
		}
		log.Println("Successfully added category column to chamas table")
	} else {
		log.Println("Column category already exists in chamas table")
	}

	// Check if target_amount column exists
	var targetAmountExists bool
	checkTargetAmountQuery := `
		SELECT COUNT(*) > 0
		FROM pragma_table_info('chamas')
		WHERE name = 'target_amount'
	`
	err = db.QueryRow(checkTargetAmountQuery).Scan(&targetAmountExists)
	if err != nil {
		return fmt.Errorf("failed to check if target_amount column exists: %w", err)
	}

	if !targetAmountExists {
		log.Println("Adding target_amount column to chamas table")
		addTargetAmountQuery := `
			ALTER TABLE chamas ADD COLUMN target_amount REAL
		`
		_, err = db.Exec(addTargetAmountQuery)
		if err != nil {
			return fmt.Errorf("failed to add target_amount column: %w", err)
		}
		log.Println("Successfully added target_amount column to chamas table")
	}

	// Check if target_deadline column exists
	var targetDeadlineExists bool
	checkTargetDeadlineQuery := `
		SELECT COUNT(*) > 0
		FROM pragma_table_info('chamas')
		WHERE name = 'target_deadline'
	`
	err = db.QueryRow(checkTargetDeadlineQuery).Scan(&targetDeadlineExists)
	if err != nil {
		return fmt.Errorf("failed to check if target_deadline column exists: %w", err)
	}

	if !targetDeadlineExists {
		log.Println("Adding target_deadline column to chamas table")
		addTargetDeadlineQuery := `
			ALTER TABLE chamas ADD COLUMN target_deadline DATETIME
		`
		_, err = db.Exec(addTargetDeadlineQuery)
		if err != nil {
			return fmt.Errorf("failed to add target_deadline column: %w", err)
		}
		log.Println("Successfully added target_deadline column to chamas table")
	}

	// Add payment method columns
	paymentColumns := []struct {
		name     string
		dataType string
	}{
		{"payment_method", "TEXT"},
		{"till_number", "TEXT"},
		{"paybill_business_number", "TEXT"},
		{"paybill_account_number", "TEXT"},
		{"payment_recipient_name", "TEXT"},
	}

	for _, col := range paymentColumns {
		var exists bool
		checkQuery := `
			SELECT COUNT(*) > 0
			FROM pragma_table_info('chamas')
			WHERE name = ?
		`
		err = db.QueryRow(checkQuery, col.name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if %s column exists: %w", col.name, err)
		}

		if !exists {
			log.Printf("Adding %s column to chamas table", col.name)
			addColumnQuery := fmt.Sprintf("ALTER TABLE chamas ADD COLUMN %s %s", col.name, col.dataType)
			_, err = db.Exec(addColumnQuery)
			if err != nil {
				return fmt.Errorf("failed to add %s column: %w", col.name, err)
			}
			log.Printf("Successfully added %s column to chamas table", col.name)
		}
	}

	return nil
}

// addMissingInvitationRoleColumns safely adds role-related columns to chama_invitations table
func addMissingInvitationRoleColumns(db *sql.DB) error {
	columns := []struct {
		name         string
		dataType     string
		defaultValue string
	}{
		{"role", "TEXT", ""},
		{"role_name", "TEXT", ""},
		{"role_description", "TEXT", ""},
	}

	for _, col := range columns {
		// Check if column exists
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('chama_invitations')
			WHERE name = ?
		`, col.name).Scan(&count)
		if err != nil {
			log.Printf("Error checking for column %s: %v", col.name, err)
			continue
		}

		// Add column if it doesn't exist
		if count == 0 {
			query := fmt.Sprintf("ALTER TABLE chama_invitations ADD COLUMN %s %s", col.name, col.dataType)
			if col.defaultValue != "" {
				query += fmt.Sprintf(" DEFAULT '%s'", col.defaultValue)
			}

			_, err = db.Exec(query)
			if err != nil {
				log.Printf("Error adding column %s: %v", col.name, err)
				continue
			}
			log.Printf("Added column %s to chama_invitations table", col.name)
		}
	}

	return nil
}

// createLearningTables creates all learning management system tables
const createLearningTables = `
-- Learning categories table
CREATE TABLE IF NOT EXISTS learning_categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    icon TEXT,
    color TEXT,
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Learning courses table
CREATE TABLE IF NOT EXISTS learning_courses (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    category_id TEXT NOT NULL,
    level TEXT NOT NULL CHECK (level IN ('beginner', 'intermediate', 'advanced')),
    type TEXT NOT NULL CHECK (type IN ('article', 'video', 'course', 'quiz')),
    content TEXT, -- Main content (markdown for articles, video URL for videos)
    thumbnail_url TEXT,
    duration_minutes INTEGER,
    estimated_read_time TEXT,
    tags TEXT, -- JSON array of tags
    prerequisites TEXT, -- JSON array of prerequisite course IDs
    learning_objectives TEXT, -- JSON array of learning objectives
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
    is_featured BOOLEAN DEFAULT false,
    -- Enhanced content fields for new system
    video_url TEXT, -- Direct video URL for video courses
    quiz_questions TEXT, -- JSON array of quiz questions with answers
    article_content TEXT, -- JSON object with headline_image and sections
    course_structure TEXT, -- JSON object with topics, subtopics, and outline
    view_count INTEGER DEFAULT 0,
    rating REAL DEFAULT 0,
    total_ratings INTEGER DEFAULT 0,
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (category_id) REFERENCES learning_categories(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Learning course lessons table (for multi-lesson courses)
CREATE TABLE IF NOT EXISTS learning_lessons (
    id TEXT PRIMARY KEY,
    course_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    lesson_order INTEGER NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('text', 'video', 'quiz', 'assignment')),
    duration_minutes INTEGER,
    video_url TEXT,
    attachments TEXT, -- JSON array of attachment URLs
    is_required BOOLEAN DEFAULT true,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (course_id) REFERENCES learning_courses(id) ON DELETE CASCADE,
    UNIQUE(course_id, lesson_order)
);

-- User course progress table
CREATE TABLE IF NOT EXISTS user_course_progress (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    course_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'not_started' CHECK (status IN ('not_started', 'in_progress', 'completed')),
    progress_percentage REAL DEFAULT 0,
    current_lesson_id TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    last_accessed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    time_spent_minutes INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (course_id) REFERENCES learning_courses(id) ON DELETE CASCADE,
    FOREIGN KEY (current_lesson_id) REFERENCES learning_lessons(id),
    UNIQUE(user_id, course_id)
);

-- User lesson progress table
CREATE TABLE IF NOT EXISTS user_lesson_progress (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    lesson_id TEXT NOT NULL,
    course_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'not_started' CHECK (status IN ('not_started', 'in_progress', 'completed')),
    started_at DATETIME,
    completed_at DATETIME,
    time_spent_minutes INTEGER DEFAULT 0,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (lesson_id) REFERENCES learning_lessons(id) ON DELETE CASCADE,
    FOREIGN KEY (course_id) REFERENCES learning_courses(id) ON DELETE CASCADE,
    UNIQUE(user_id, lesson_id)
);

-- Course ratings and reviews table
CREATE TABLE IF NOT EXISTS learning_course_reviews (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    course_id TEXT NOT NULL,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review TEXT,
    is_verified BOOLEAN DEFAULT false, -- true if user completed the course
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (course_id) REFERENCES learning_courses(id) ON DELETE CASCADE,
    UNIQUE(user_id, course_id)
);

-- Learning achievements/certificates table
CREATE TABLE IF NOT EXISTS learning_achievements (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    course_id TEXT NOT NULL,
    achievement_type TEXT NOT NULL CHECK (achievement_type IN ('completion', 'excellence', 'speed', 'consistency')),
    title TEXT NOT NULL,
    description TEXT,
    badge_url TEXT,
    certificate_url TEXT,
    earned_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (course_id) REFERENCES learning_courses(id) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_learning_courses_category ON learning_courses(category_id);
CREATE INDEX IF NOT EXISTS idx_learning_courses_status ON learning_courses(status);
CREATE INDEX IF NOT EXISTS idx_learning_courses_level ON learning_courses(level);
CREATE INDEX IF NOT EXISTS idx_learning_courses_type ON learning_courses(type);
CREATE INDEX IF NOT EXISTS idx_learning_courses_featured ON learning_courses(is_featured);
CREATE INDEX IF NOT EXISTS idx_learning_lessons_course ON learning_lessons(course_id);
CREATE INDEX IF NOT EXISTS idx_user_course_progress_user ON user_course_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_user_course_progress_course ON user_course_progress(course_id);
CREATE INDEX IF NOT EXISTS idx_user_lesson_progress_user ON user_lesson_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_user_lesson_progress_lesson ON user_lesson_progress(lesson_id);
CREATE INDEX IF NOT EXISTS idx_learning_course_reviews_course ON learning_course_reviews(course_id);
CREATE INDEX IF NOT EXISTS idx_learning_achievements_user ON learning_achievements(user_id);
`

const createRemindersTable = `
CREATE TABLE IF NOT EXISTS reminders (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    reminder_type TEXT NOT NULL DEFAULT 'once', -- 'once', 'daily', 'weekly', 'monthly'
    scheduled_at DATETIME NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE,
    is_completed BOOLEAN DEFAULT FALSE,
    notification_sent BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_reminders_user_id ON reminders(user_id);
CREATE INDEX IF NOT EXISTS idx_reminders_scheduled_at ON reminders(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_reminders_type ON reminders(reminder_type);
CREATE INDEX IF NOT EXISTS idx_reminders_enabled ON reminders(is_enabled);
CREATE INDEX IF NOT EXISTS idx_reminders_completed ON reminders(is_completed);
`

const createSharesAndDividendsTables = `
-- Shares ownership table
CREATE TABLE IF NOT EXISTS shares (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    member_id TEXT NOT NULL,
    name TEXT NOT NULL,
    share_type TEXT NOT NULL DEFAULT 'ordinary', -- 'ordinary', 'preferred'
    shares_owned INTEGER NOT NULL DEFAULT 0,
    share_value REAL NOT NULL DEFAULT 0,
    total_value REAL NOT NULL DEFAULT 0,
    purchase_date DATETIME NOT NULL,
    certificate_number TEXT,
    status TEXT NOT NULL DEFAULT 'active', -- 'active', 'transferred', 'redeemed'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (member_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Dividend declarations table
CREATE TABLE IF NOT EXISTS dividend_declarations (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    declaration_date DATETIME,
    dividend_per_share REAL NOT NULL,
    total_amount REAL NOT NULL,
    payment_date DATETIME,
    status TEXT NOT NULL DEFAULT 'declared', -- 'declared', 'approved', 'paid', 'cancelled'
    declared_by TEXT,
    approved_by TEXT,
    description TEXT,
    dividend_type TEXT DEFAULT 'cash',
    eligibility_criteria TEXT,
    approval_required BOOLEAN DEFAULT TRUE,
    created_by TEXT,
    created_by_id TEXT,
    timestamp DATETIME,
    transaction_id TEXT,
    security_hash TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (declared_by) REFERENCES users(id),
    FOREIGN KEY (approved_by) REFERENCES users(id),
    FOREIGN KEY (created_by_id) REFERENCES users(id)
);

-- Individual dividend payments table
CREATE TABLE IF NOT EXISTS dividend_payments (
    id TEXT PRIMARY KEY,
    dividend_declaration_id TEXT NOT NULL,
    member_id TEXT NOT NULL,
    shares_eligible INTEGER NOT NULL,
    dividend_amount REAL NOT NULL,
    payment_status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'paid', 'failed'
    payment_date DATETIME,
    payment_method TEXT,
    transaction_reference TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (dividend_declaration_id) REFERENCES dividend_declarations(id) ON DELETE CASCADE,
    FOREIGN KEY (member_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Share transactions table (for transfers, purchases, redemptions)
CREATE TABLE IF NOT EXISTS share_transactions (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    from_member_id TEXT,
    to_member_id TEXT,
    transaction_type TEXT NOT NULL, -- 'purchase', 'transfer', 'redemption', 'split'
    shares_count INTEGER NOT NULL,
    share_value REAL NOT NULL,
    total_amount REAL NOT NULL,
    transaction_date DATETIME NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'completed', 'cancelled'
    approved_by TEXT,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (from_member_id) REFERENCES users(id),
    FOREIGN KEY (to_member_id) REFERENCES users(id),
    FOREIGN KEY (approved_by) REFERENCES users(id)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_shares_chama_member ON shares(chama_id, member_id);
CREATE INDEX IF NOT EXISTS idx_shares_status ON shares(status);
CREATE INDEX IF NOT EXISTS idx_dividend_declarations_chama ON dividend_declarations(chama_id);
CREATE INDEX IF NOT EXISTS idx_dividend_declarations_status ON dividend_declarations(status);
CREATE INDEX IF NOT EXISTS idx_dividend_payments_declaration ON dividend_payments(dividend_declaration_id);
CREATE INDEX IF NOT EXISTS idx_dividend_payments_member ON dividend_payments(member_id);
CREATE INDEX IF NOT EXISTS idx_dividend_payments_status ON dividend_payments(payment_status);
CREATE INDEX IF NOT EXISTS idx_share_transactions_chama ON share_transactions(chama_id);
CREATE INDEX IF NOT EXISTS idx_share_transactions_type ON share_transactions(transaction_type);
CREATE INDEX IF NOT EXISTS idx_share_transactions_status ON share_transactions(status);
`

const createPollsAndVotingTables = `
-- Polls table for voting and role escalation
CREATE TABLE IF NOT EXISTS polls (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    poll_type TEXT NOT NULL, -- 'general', 'Election / Voting', 'financial_decision'
    created_by TEXT NOT NULL,
    start_date DATETIME NOT NULL,
    end_date DATETIME NOT NULL,
    status TEXT NOT NULL DEFAULT 'active', -- 'active', 'completed', 'cancelled'
    is_anonymous BOOLEAN DEFAULT TRUE,
    requires_majority BOOLEAN DEFAULT TRUE,
    majority_percentage REAL DEFAULT 50.0,
    total_eligible_voters INTEGER DEFAULT 0,
    total_votes_cast INTEGER DEFAULT 0,
    result TEXT, -- 'passed', 'failed', 'pending'
    result_declared_at DATETIME,
    metadata TEXT, -- JSON for additional poll-specific data
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Poll options table
CREATE TABLE IF NOT EXISTS poll_options (
    id TEXT PRIMARY KEY,
    poll_id TEXT NOT NULL,
    option_text TEXT NOT NULL,
    option_order INTEGER NOT NULL DEFAULT 0,
    vote_count INTEGER DEFAULT 0,
    metadata TEXT, -- JSON for option-specific data (e.g., candidate info for role escalation)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (poll_id) REFERENCES polls(id) ON DELETE CASCADE
);

-- Votes table (anonymous voting)
CREATE TABLE IF NOT EXISTS votes (
    id TEXT PRIMARY KEY,
    poll_id TEXT NOT NULL,
    option_id TEXT NOT NULL,
    voter_hash TEXT NOT NULL, -- Hashed voter ID for anonymity
    vote_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_valid BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (poll_id) REFERENCES polls(id) ON DELETE CASCADE,
    FOREIGN KEY (option_id) REFERENCES poll_options(id) ON DELETE CASCADE,
    UNIQUE(poll_id, voter_hash) -- One vote per voter per poll
);

-- Role escalation requests table
CREATE TABLE IF NOT EXISTS Election / Voting_requests (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    candidate_id TEXT NOT NULL,
    current_role TEXT NOT NULL,
    requested_role TEXT NOT NULL,
    requested_by TEXT NOT NULL,
    poll_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'rejected', 'voting'
    justification TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (candidate_id) REFERENCES users(id),
    FOREIGN KEY (requested_by) REFERENCES users(id),
    FOREIGN KEY (poll_id) REFERENCES polls(id)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_polls_chama ON polls(chama_id);
CREATE INDEX IF NOT EXISTS idx_polls_status ON polls(status);
CREATE INDEX IF NOT EXISTS idx_polls_type ON polls(poll_type);
CREATE INDEX IF NOT EXISTS idx_polls_dates ON polls(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_poll_options_poll ON poll_options(poll_id);
CREATE INDEX IF NOT EXISTS idx_votes_poll ON votes(poll_id);
CREATE INDEX IF NOT EXISTS idx_votes_option ON votes(option_id);
CREATE INDEX IF NOT EXISTS idx_votes_hash ON votes(voter_hash);
CREATE INDEX IF NOT EXISTS idx_Election / Voting_chama ON Election / Voting_requests(chama_id);
CREATE INDEX IF NOT EXISTS idx_Election / Voting_candidate ON Election / Voting_requests(candidate_id);
CREATE INDEX IF NOT EXISTS idx_Election / Voting_status ON Election / Voting_requests(status);
`

const createDisbursementTables = `
-- Disbursement batches table (for mass distributions)
CREATE TABLE IF NOT EXISTS disbursement_batches (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    batch_type TEXT NOT NULL, -- 'dividend', 'shares', 'savings', 'loan'
    title TEXT NOT NULL,
    description TEXT,
    total_amount REAL NOT NULL,
    total_recipients INTEGER NOT NULL,
    initiated_by TEXT NOT NULL,
    approved_by TEXT,
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'processing', 'completed', 'failed'
    approval_required BOOLEAN DEFAULT TRUE,
    scheduled_date DATETIME,
    processed_date DATETIME,
    metadata TEXT, -- JSON for batch-specific data
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (initiated_by) REFERENCES users(id),
    FOREIGN KEY (approved_by) REFERENCES users(id)
);

-- Individual disbursements table
CREATE TABLE IF NOT EXISTS disbursements (
    id TEXT PRIMARY KEY,
    batch_id TEXT,
    recipient_id TEXT NOT NULL,
    disbursement_type TEXT NOT NULL, -- 'dividend', 'share_redemption', 'savings_withdrawal', 'loan_disbursement', 'shares', 'savings_withdrawal', 'other'
    amount REAL NOT NULL,
    currency TEXT DEFAULT 'KES',
    payment_method TEXT NOT NULL, -- 'bank_transfer', 'mobile_money', 'cash'
    account_details TEXT, -- JSON with payment details
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    transaction_reference TEXT,
    processed_date DATETIME,
    failure_reason TEXT,
    retry_count INTEGER DEFAULT 0,
    metadata TEXT, -- JSON for disbursement-specific data
    chama_id TEXT NOT NULL,
    initiated_by TEXT NOT NULL,
    initiated_by_id TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    transaction_id TEXT NOT NULL,
    security_hash TEXT NOT NULL,
    purpose TEXT,
    private_note TEXT,
    from_account TEXT,
    to_account TEXT,
    member_name TEXT,
    member_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (batch_id) REFERENCES disbursement_batches(id) ON DELETE CASCADE,
    FOREIGN KEY (recipient_id) REFERENCES users(id),
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (initiated_by_id) REFERENCES users(id)
);

-- Bulk disbursements table for dividends
CREATE TABLE IF NOT EXISTS bulk_disbursements (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    type TEXT NOT NULL,
    category TEXT NOT NULL,
    dividend_per_share REAL,
    total_amount REAL NOT NULL,
    description TEXT,
    from_account TEXT NOT NULL,
    initiated_by TEXT NOT NULL,
    initiated_by_id TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    status TEXT NOT NULL,
    transaction_id TEXT NOT NULL,
    security_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (initiated_by_id) REFERENCES users(id)
);

-- Dividends table for individual dividend payments
CREATE TABLE IF NOT EXISTS dividends (
    id TEXT PRIMARY KEY,
    bulk_disbursement_id TEXT,
    chama_id TEXT NOT NULL,
    member_id TEXT NOT NULL,
    member_name TEXT NOT NULL,
    shares_owned INTEGER NOT NULL,
    dividend_per_share REAL NOT NULL,
    amount REAL NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bulk_disbursement_id) REFERENCES bulk_disbursements(id) ON DELETE CASCADE,
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (member_id) REFERENCES users(id)
);

-- Share offerings table
CREATE TABLE IF NOT EXISTS share_offerings (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    name TEXT NOT NULL,
    share_type TEXT NOT NULL,
    total_shares INTEGER NOT NULL,
    price_per_share REAL NOT NULL,
    minimum_purchase INTEGER DEFAULT 1,
    description TEXT,
    eligibility_criteria TEXT,
    approval_required BOOLEAN DEFAULT FALSE,
    total_value REAL NOT NULL,
    created_by TEXT NOT NULL,
    created_by_id TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    transaction_id TEXT NOT NULL,
    security_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by_id) REFERENCES users(id)
);

-- Financial transparency log table
CREATE TABLE IF NOT EXISTS financial_transparency_log (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    activity_type TEXT NOT NULL, -- 'disbursement', 'revenue', 'expense', 'contribution'
    title TEXT NOT NULL,
    description TEXT,
    amount REAL NOT NULL,
    currency TEXT DEFAULT 'KES',
    transaction_type TEXT NOT NULL, -- 'debit', 'credit'
    reference_id TEXT, -- Reference to related transaction/disbursement
    reference_type TEXT, -- 'disbursement_batch', 'transaction', 'contribution'
    performed_by TEXT NOT NULL,
    affected_members TEXT, -- JSON array of affected member IDs
    visibility TEXT NOT NULL DEFAULT 'all_members', -- 'all_members', 'officials_only'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (performed_by) REFERENCES users(id)
);

-- Reports and invoices table
CREATE TABLE IF NOT EXISTS financial_reports (
    id TEXT PRIMARY KEY,
    chama_id TEXT NOT NULL,
    report_type TEXT NOT NULL, -- 'monthly_statement', 'dividend_report', 'disbursement_report', 'transparency_report'
    title TEXT NOT NULL,
    description TEXT,
    report_period_start DATETIME,
    report_period_end DATETIME,
    generated_by TEXT NOT NULL,
    file_path TEXT, -- Path to generated PDF/document
    file_size INTEGER,
    status TEXT NOT NULL DEFAULT 'generating', -- 'generating', 'ready', 'failed'
    download_count INTEGER DEFAULT 0,
    is_public BOOLEAN DEFAULT FALSE,
    metadata TEXT, -- JSON for report-specific data
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chama_id) REFERENCES chamas(id) ON DELETE CASCADE,
    FOREIGN KEY (generated_by) REFERENCES users(id)
);`

// addEnhancedLearningContentFields adds enhanced content fields to learning_courses table
const addEnhancedLearningContentFields = "SELECT 1" // Placeholder - actual logic in migration function

// createQuizResultsTable creates the quiz_results table for storing quiz results
const createQuizResultsTable = `
CREATE TABLE IF NOT EXISTS quiz_results (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id TEXT NOT NULL,
    course_id TEXT NOT NULL,
    score INTEGER NOT NULL,
    correct_answers INTEGER NOT NULL,
    total_questions INTEGER NOT NULL,
    passed BOOLEAN NOT NULL DEFAULT 0,
    time_taken INTEGER, -- in seconds
    detailed_results TEXT, -- JSON string with detailed results
    created_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (course_id) REFERENCES learning_courses(id) ON DELETE CASCADE
);`

// addMissingEnhancedLearningContentFields adds enhanced learning content fields if they don't exist
func addMissingEnhancedLearningContentFields(db *sql.DB) error {
	// First check if the learning_courses table exists
	var tableExists bool
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='learning_courses'`
	err := db.QueryRow(query).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check if learning_courses table exists: %w", err)
	}

	if !tableExists {
		log.Printf("learning_courses table does not exist, skipping enhanced learning content fields migration")
		return nil
	}

	// Define the columns to add
	columns := []struct {
		name     string
		dataType string
	}{
		{"video_url", "TEXT"},
		{"quiz_questions", "TEXT"},
		{"article_content", "TEXT"},
		{"course_structure", "TEXT"},
	}

	// Check and add each column
	for _, col := range columns {
		var exists bool
		query := `SELECT COUNT(*) FROM pragma_table_info('learning_courses') WHERE name = ?`
		err := db.QueryRow(query, col.name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if column %s exists: %w", col.name, err)
		}

		// Add column if it doesn't exist
		if !exists {
			alterQuery := fmt.Sprintf("ALTER TABLE learning_courses ADD COLUMN %s %s", col.name, col.dataType)
			if _, err := db.Exec(alterQuery); err != nil {
				return fmt.Errorf("failed to add column %s: %w", col.name, err)
			}
			log.Printf("Added column %s to learning_courses table", col.name)
		} else {
			log.Printf("Column %s already exists in learning_courses table", col.name)
		}
	}

	return nil
}

// addChamaCategoryColumn adds category column to chamas table
const addChamaCategoryColumn = `
-- This will be handled by addCategoryColumnToChamasTable function
SELECT 1;
`

// MigrationManager handles database migrations
type MigrationManager struct {
	db *sql.DB
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *sql.DB) *MigrationManager {
	return &MigrationManager{db: db}
}

// RunMigrations executes all pending migrations
func (m *MigrationManager) RunMigrations() error {
	log.Println("🚀 Starting database migrations...")

	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Run notification system migration
	if err := m.runMigration("create_notification_system_tables", m.createNotificationSystemTables); err != nil {
		return fmt.Errorf("failed to run notification system migration: %w", err)
	}

	// Insert default data
	if err := m.runMigration("insert_default_notification_data", m.insertDefaultNotificationData); err != nil {
		return fmt.Errorf("failed to insert default notification data: %w", err)
	}

	// Insert vibrate sound if it is missing
	if err := m.runMigration("insert_vibrate_sound", m.insertVibrateSound); err != nil {
		return fmt.Errorf("failed to insert vibrate sound: %w", err)
	}

	log.Println("✅ All migrations completed successfully!")
	return nil
}

// insertVibrateSound inserts the 'Vibrate' notification sound if it doesn't exist.
func (m *MigrationManager) insertVibrateSound() error {
	// Check if 'Vibrate' sound already exists
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM notification_sounds WHERE name = ?", "Vibrate").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Println("ℹ️  'Vibrate' notification sound already exists, skipping insertion.")
		return nil
	}

	// 'Vibrate' sound does not exist, so insert it
	log.Println("🎶 Inserting 'Vibrate' notification sound...")
	stmt, err := m.db.Prepare(`
		INSERT INTO notification_sounds (name, file_path, is_default, is_active, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	_, err = stmt.Exec("Vibrate", "/notification_sound/vibrate.mp3", false, true, now, now)
	if err != nil {
		return err
	}

	log.Println("✅ 'Vibrate' notification sound inserted successfully!")
	return nil
}

// createMigrationsTable creates the migrations tracking table
func (m *MigrationManager) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			migration VARCHAR(255) NOT NULL UNIQUE,
			executed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := m.db.Exec(query)
	return err
}

// runMigration executes a migration if it hasn't been run before
func (m *MigrationManager) runMigration(name string, migrationFunc func() error) error {
	// Check if migration has already been run
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM migrations WHERE migration = ?", name).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Printf("⏭️  Migration '%s' already executed, skipping...", name)
		return nil
	}

	log.Printf("🔄 Running migration: %s", name)

	// Start transaction
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration
	if err := migrationFunc(); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Record migration as completed
	_, err = tx.Exec("INSERT INTO migrations (migration) VALUES (?)", name)
	if err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("✅ Migration '%s' completed successfully!", name)
	return nil
}

// createNotificationSystemTables creates all notification system tables
func (m *MigrationManager) createNotificationSystemTables() error {
	migrations := []string{
		// notification_sounds table
		`CREATE TABLE IF NOT EXISTS notification_sounds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(100) NOT NULL,
			file_path VARCHAR(255) NOT NULL,
			file_size INTEGER DEFAULT 0,
			duration_seconds REAL DEFAULT 0.00,
			is_default BOOLEAN DEFAULT 0,
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// user_notification_preferences table
		`CREATE TABLE IF NOT EXISTS user_notification_preferences (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			notification_sound_id INTEGER DEFAULT NULL,
			sound_enabled BOOLEAN DEFAULT 1,
			vibration_enabled BOOLEAN DEFAULT 1,
			volume_level INTEGER DEFAULT 80 CHECK (volume_level BETWEEN 0 AND 100),
			chama_notifications BOOLEAN DEFAULT 1,
			transaction_notifications BOOLEAN DEFAULT 1,
			reminder_notifications BOOLEAN DEFAULT 1,
			system_notifications BOOLEAN DEFAULT 1,
			marketing_notifications BOOLEAN DEFAULT 0,
			quiet_hours_enabled BOOLEAN DEFAULT 0,
			quiet_hours_start TIME DEFAULT '22:00:00',
			quiet_hours_end TIME DEFAULT '07:00:00',
			timezone VARCHAR(50) DEFAULT 'Africa/Nairobi',
			notification_frequency VARCHAR(20) DEFAULT 'immediate' CHECK (notification_frequency IN ('immediate', 'batched_15min', 'batched_1hour', 'daily_digest')),
			priority_only_during_quiet BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (notification_sound_id) REFERENCES notification_sounds(id) ON DELETE SET NULL,
			UNIQUE(user_id)
		)`,

		// Enhanced notifications table (add columns if table exists)
		`CREATE TABLE IF NOT EXISTS notifications_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			title VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			type VARCHAR(20) NOT NULL CHECK (type IN ('chama', 'transaction', 'reminder', 'system', 'marketing', 'alert')),
			priority VARCHAR(10) DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
			category VARCHAR(100) DEFAULT NULL,
			reference_type VARCHAR(50) DEFAULT NULL,
			reference_id INTEGER DEFAULT NULL,
			status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'delivered', 'read', 'failed')),
			is_read BOOLEAN DEFAULT 0,
			read_at DATETIME NULL,
			scheduled_for DATETIME DEFAULT CURRENT_TIMESTAMP,
			sent_at DATETIME NULL,
			delivered_at DATETIME NULL,
			data TEXT DEFAULT NULL,
			sound_played BOOLEAN DEFAULT 0,
			retry_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// notification_templates table
		`CREATE TABLE IF NOT EXISTS notification_templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(100) NOT NULL UNIQUE,
			type VARCHAR(20) NOT NULL CHECK (type IN ('chama', 'transaction', 'reminder', 'system', 'marketing', 'alert')),
			category VARCHAR(100) NOT NULL,
			title_template VARCHAR(255) NOT NULL,
			message_template TEXT NOT NULL,
			default_priority VARCHAR(10) DEFAULT 'normal' CHECK (default_priority IN ('low', 'normal', 'high', 'urgent')),
			requires_sound BOOLEAN DEFAULT 1,
			requires_vibration BOOLEAN DEFAULT 1,
			variables TEXT DEFAULT NULL,
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// notification_delivery_log table
		`CREATE TABLE IF NOT EXISTS notification_delivery_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			notification_id INTEGER NOT NULL,
			user_id TEXT NOT NULL,
			delivery_method VARCHAR(20) NOT NULL CHECK (delivery_method IN ('push', 'sms', 'email', 'in_app')),
			status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'bounced')),
			attempted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			delivered_at DATETIME NULL,
			error_message TEXT NULL,
			retry_count INTEGER DEFAULT 0,
			device_info TEXT DEFAULT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (notification_id) REFERENCES notifications(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// user_reminders table
		`CREATE TABLE IF NOT EXISTS user_reminders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			title VARCHAR(255) NOT NULL,
			description TEXT DEFAULT NULL,
			reminder_datetime DATETIME NOT NULL,
			timezone VARCHAR(50) DEFAULT 'Africa/Nairobi',
			is_recurring BOOLEAN DEFAULT 0,
			recurrence_pattern VARCHAR(20) DEFAULT NULL CHECK (recurrence_pattern IN ('daily', 'weekly', 'monthly', 'yearly')),
			recurrence_interval INTEGER DEFAULT 1,
			recurrence_end_date DATE DEFAULT NULL,
			sound_enabled BOOLEAN DEFAULT 1,
			vibration_enabled BOOLEAN DEFAULT 1,
			custom_sound_id INTEGER DEFAULT NULL,
			status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'completed', 'cancelled', 'snoozed')),
			snooze_until DATETIME NULL,
			completed_at DATETIME NULL,
			category VARCHAR(100) DEFAULT 'personal',
			priority VARCHAR(10) DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (custom_sound_id) REFERENCES notification_sounds(id) ON DELETE SET NULL
		)`,
	}

	// Execute all migrations
	for _, migration := range migrations {
		if _, err := m.db.Exec(migration); err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}

	// Handle existing notifications table migration
	if err := m.migrateExistingNotificationsTable(); err != nil {
		return fmt.Errorf("failed to migrate existing notifications table: %w", err)
	}

	// Create indexes
	if err := m.createIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// migrateExistingNotificationsTable handles migration of existing notifications table
func (m *MigrationManager) migrateExistingNotificationsTable() error {
	// Check if old notifications table exists
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='notifications'").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		// Get existing table structure
		rows, err := m.db.Query("PRAGMA table_info(notifications)")
		if err != nil {
			return err
		}
		defer rows.Close()

		existingColumns := make(map[string]bool)
		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var defaultValue sql.NullString

			err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
			if err != nil {
				return err
			}
			existingColumns[name] = true
		}

		// Build SELECT query based on existing columns (user_id is already TEXT)
		selectParts := []string{
			"CAST(id AS INTEGER) as id",
			"user_id",
			"title",
			"message",
			"CASE WHEN type IS NULL THEN 'system' ELSE type END as type",
			"COALESCE(is_read, 0) as is_read",
		}

		// Add read_at if it exists, otherwise NULL
		if existingColumns["read_at"] {
			selectParts = append(selectParts, "read_at")
		} else {
			selectParts = append(selectParts, "NULL as read_at")
		}

		selectParts = append(selectParts, "created_at")

		// Add updated_at if it exists, otherwise use created_at
		if existingColumns["updated_at"] {
			selectParts = append(selectParts, "COALESCE(updated_at, created_at) as updated_at")
		} else {
			selectParts = append(selectParts, "created_at as updated_at")
		}

		selectQuery := strings.Join(selectParts, ", ")

		// Copy data from old table to new table
		copyQuery := fmt.Sprintf(`
			INSERT OR IGNORE INTO notifications_new
			(id, user_id, title, message, type, is_read, read_at, created_at, updated_at)
			SELECT %s FROM notifications
		`, selectQuery)

		_, err = m.db.Exec(copyQuery)
		if err != nil {
			return fmt.Errorf("failed to copy data: %w", err)
		}

		// Drop old table
		if _, err = m.db.Exec("DROP TABLE notifications"); err != nil {
			return fmt.Errorf("failed to drop old table: %w", err)
		}

		log.Println("✅ Migrated existing notifications table data")
	}

	// Rename new table to notifications
	_, err = m.db.Exec("ALTER TABLE notifications_new RENAME TO notifications")
	if err != nil {
		// If rename fails, the table might already be named correctly
		log.Println("Note: notifications table rename failed, might already be correct")
	}

	return nil
}

// createIndexes creates database indexes for better performance
func (m *MigrationManager) createIndexes() error {
	indexes := []string{
		// notification_sounds indexes
		"CREATE INDEX IF NOT EXISTS idx_notification_sounds_is_default ON notification_sounds(is_default)",
		"CREATE INDEX IF NOT EXISTS idx_notification_sounds_is_active ON notification_sounds(is_active)",

		// user_notification_preferences indexes
		"CREATE INDEX IF NOT EXISTS idx_user_notification_preferences_user_id ON user_notification_preferences(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_user_notification_preferences_sound_enabled ON user_notification_preferences(sound_enabled)",

		// notifications indexes
		"CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_priority ON notifications(priority)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_scheduled_for ON notifications(scheduled_for)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(is_read)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, is_read)",

		// notification_templates indexes
		"CREATE INDEX IF NOT EXISTS idx_notification_templates_type_category ON notification_templates(type, category)",
		"CREATE INDEX IF NOT EXISTS idx_notification_templates_is_active ON notification_templates(is_active)",

		// notification_delivery_log indexes
		"CREATE INDEX IF NOT EXISTS idx_notification_delivery_log_notification_id ON notification_delivery_log(notification_id)",
		"CREATE INDEX IF NOT EXISTS idx_notification_delivery_log_user_id ON notification_delivery_log(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_notification_delivery_log_status ON notification_delivery_log(status)",

		// user_reminders indexes
		"CREATE INDEX IF NOT EXISTS idx_user_reminders_user_id ON user_reminders(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_user_reminders_reminder_datetime ON user_reminders(reminder_datetime)",
		"CREATE INDEX IF NOT EXISTS idx_user_reminders_status ON user_reminders(status)",
		"CREATE INDEX IF NOT EXISTS idx_user_reminders_user_active ON user_reminders(user_id, status, reminder_datetime)",
	}

	for _, index := range indexes {
		if _, err := m.db.Exec(index); err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
		}
	}

	return nil
}

// insertDefaultNotificationData inserts default sounds and templates
func (m *MigrationManager) insertDefaultNotificationData() error {
	// Insert default notification sounds
	if err := m.insertDefaultSounds(); err != nil {
		return fmt.Errorf("failed to insert default sounds: %w", err)
	}

	// Insert default notification templates
	if err := m.insertDefaultTemplates(); err != nil {
		return fmt.Errorf("failed to insert default templates: %w", err)
	}

	return nil
}

// insertDefaultSounds inserts or updates default notification sounds to ensure they are always up-to-date.
func (m *MigrationManager) insertDefaultSounds() error {
	sounds := []struct {
		name      string
		filePath  string
		isDefault bool
		isActive  bool
	}{
		{"Default Ring", "/notification_sound/ring.mp3", true, true},
		{"Gentle Bell", "/notification_sound/bell.mp3", false, true},
		{"Alert Tone", "/notification_sound/alert.mp3", false, true},
		{"Chime", "/notification_sound/chime.mp3", false, true},
		{"Vibrate", "/notification_sound/vibrate.mp3", false, true},
		{"Silent", "", false, true},
	}

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback on error, no-op on success

	now := time.Now()

	for _, sound := range sounds {
		var id int
		err := tx.QueryRow("SELECT id FROM notification_sounds WHERE name = ?", sound.name).Scan(&id)
		if err != nil {
			if err == sql.ErrNoRows {
				// Insert
				_, err = tx.Exec(`
					INSERT INTO notification_sounds (name, file_path, is_default, is_active, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?)
				`, sound.name, sound.filePath, sound.isDefault, sound.isActive, now, now)
				if err != nil {
					return fmt.Errorf("failed to insert sound %s: %w", sound.name, err)
				}
			} else {
				return fmt.Errorf("failed to query for sound %s: %w", sound.name, err)
			}
		} else {
			// Update
			_, err = tx.Exec(`
				UPDATE notification_sounds
				SET file_path = ?, is_default = ?, is_active = ?, updated_at = ?
				WHERE id = ?
			`, sound.filePath, sound.isDefault, sound.isActive, now, id)
			if err != nil {
				return fmt.Errorf("failed to update sound %s: %w", sound.name, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Println("✅ Default notification sounds successfully upserted!")
	return nil
}

// insertDefaultTemplates inserts default notification templates
func (m *MigrationManager) insertDefaultTemplates() error {
	// Check if templates already exist
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM notification_templates").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Println("ℹ️  Notification templates already exist, skipping default template insertion.")
		return nil
	}

	templates := []struct {
		name            string
		notifType       string
		category        string
		titleTemplate   string
		messageTemplate string
		defaultPriority string
		variables       string
	}{
		{
			"chama_payment_due", "chama", "payment_due",
			"Payment Due: {chama_name}",
			"Your contribution of KSh {amount} for {chama_name} is due on {due_date}.",
			"high", `["chama_name", "amount", "due_date"]`,
		},
		{
			"transaction_received", "transaction", "deposit_received",
			"Payment Received",
			"You have received KSh {amount} from {sender_name}.",
			"normal", `["amount", "sender_name"]`,
		},
		{
			"meeting_reminder", "reminder", "meeting",
			"Meeting Reminder: {chama_name}",
			"Don't forget about the {chama_name} meeting scheduled for {meeting_time}.",
			"high", `["chama_name", "meeting_time"]`,
		},
		{
			"system_maintenance", "system", "maintenance",
			"System Maintenance",
			"VaultKe will undergo maintenance from {start_time} to {end_time}. Some features may be unavailable.",
			"normal", `["start_time", "end_time"]`,
		},
	}

	stmt, err := m.db.Prepare(`
		INSERT INTO notification_templates 
		(name, type, category, title_template, message_template, default_priority, variables, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, template := range templates {
		_, err = stmt.Exec(
			template.name, template.notifType, template.category,
			template.titleTemplate, template.messageTemplate,
			template.defaultPriority, template.variables, now, now,
		)
		if err != nil {
			return err
		}
	}

	log.Println("✅ Default notification templates inserted successfully!")
	return nil
}

// GetMigrationStatus returns the status of all migrations
func (m *MigrationManager) GetMigrationStatus() ([]map[string]interface{}, error) {
	rows, err := m.db.Query("SELECT migration, executed_at FROM migrations ORDER BY executed_at ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []map[string]interface{}
	for rows.Next() {
		var migration string
		var executedAt time.Time

		if err := rows.Scan(&migration, &executedAt); err != nil {
			return nil, err
		}

		migrations = append(migrations, map[string]interface{}{
			"migration":   migration,
			"executed_at": executedAt,
		})
	}

	return migrations, nil
}

// IsNotificationSystemReady checks if the notification system is properly set up
func (m *MigrationManager) IsNotificationSystemReady() (bool, error) {
	// Check if default sound exists
	var defaultSoundCount int
	err := m.db.QueryRow("SELECT COUNT(*) FROM notification_sounds WHERE is_default = 1").Scan(&defaultSoundCount)
	if err != nil {
		return false, err
	}

	// Check if templates exist
	var templateCount int
	err = m.db.QueryRow("SELECT COUNT(*) FROM notification_templates").Scan(&templateCount)
	if err != nil {
		return false, err
	}

	return defaultSoundCount > 0 && templateCount > 0, nil
}

// updateChatMessageStorage updates the chat_messages table to optimize message storage
func updateChatMessageStorage(db *sql.DB) error {
	// Add encryption_metadata column
	var exists bool
	query := `SELECT COUNT(*) FROM pragma_table_info('chat_messages') WHERE name = 'encryption_metadata'`
	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if encryption_metadata column exists: %w", err)
	}

	if !exists {
		alterQuery := `ALTER TABLE chat_messages ADD COLUMN encryption_metadata TEXT DEFAULT '{}'`
		if _, err := db.Exec(alterQuery); err != nil {
			return fmt.Errorf("failed to add encryption_metadata column: %w", err)
		}
		log.Printf("Added encryption_metadata column to chat_messages table")
	}

	// Update existing records
	rows, err := db.Query(`SELECT id, message FROM chat_messages`)
	if err != nil {
		return fmt.Errorf("failed to query chat messages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var message string
		if err := rows.Scan(&id, &message); err != nil {
			return err
		}

		// Parse the message JSON
		var msgData map[string]interface{}
		if err := json.Unmarshal([]byte(message), &msgData); err != nil {
			// If not JSON, perhaps it's already plain, skip
			continue
		}

		if ciphertext, ok := msgData["ciphertext"]; ok {
			// Encrypted message
			// Set message to ciphertext
			// Set encryption_metadata to the rest
			delete(msgData, "ciphertext")
			if _, hasContent := msgData["content"]; hasContent {
				delete(msgData, "content")
			}
			metaBytes, _ := json.Marshal(msgData)
			updateQuery := `UPDATE chat_messages SET message = ?, encryption_metadata = ? WHERE id = ?`
			if _, err := db.Exec(updateQuery, ciphertext, string(metaBytes), id); err != nil {
				return fmt.Errorf("failed to update message %s: %w", id, err)
			}
		} else if content, ok := msgData["content"]; ok {
			// Plain text
			// Set message to content
			// encryption_metadata remains '{}'
			updateQuery := `UPDATE chat_messages SET message = ? WHERE id = ?`
			if _, err := db.Exec(updateQuery, content, id); err != nil {
				return fmt.Errorf("failed to update message %s: %w", id, err)
			}
		}
	}

	log.Printf("Refactored chat messages to reduce redundancy")
	return nil
}

// refactorChatMessageContent refactors chat_messages to store only ciphertext in content column
func refactorChatMessageContent(db *sql.DB) error {
	log.Printf("Starting chat message content refactor...")

	// Update existing records: parse content JSON, extract ciphertext to content, move encryption metadata to metadata
	rows, err := db.Query(`SELECT id, content FROM chat_messages WHERE content LIKE '{%'`)
	if err != nil {
		return fmt.Errorf("failed to query chat messages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var content string
		if err := rows.Scan(&id, &content); err != nil {
			return err
		}

		// Parse the content JSON
		var contentData map[string]interface{}
		if err := json.Unmarshal([]byte(content), &contentData); err != nil {
			// If not valid JSON, skip
			continue
		}

		if ciphertext, ok := contentData["ciphertext"]; ok {
			if cipherStr, ok := ciphertext.(string); ok {
				// Extract encryption metadata (everything except ciphertext)
				encryptionMeta := make(map[string]interface{})
				for k, v := range contentData {
					if k != "ciphertext" {
						encryptionMeta[k] = v
					}
				}

				metaBytes, _ := json.Marshal(encryptionMeta)
				updateQuery := `UPDATE chat_messages SET content = ?, metadata = ? WHERE id = ?`
				if _, err := db.Exec(updateQuery, cipherStr, string(metaBytes), id); err != nil {
					return fmt.Errorf("failed to update message %s: %w", id, err)
				}
			}
		}
	}

	log.Printf("Refactored chat message content to store only ciphertext")
	return nil
}

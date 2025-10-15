#!/bin/bash

# =====================================================
# VaultKe Notification System Cron Jobs
# =====================================================

# Add these lines to your server's crontab:
# Run: crontab -e
# Then add the following lines:

# Process notifications and reminders every minute
# This handles immediate notifications, due reminders, and failed notification retries
* * * * * cd /path/to/vaultke/backend && php artisan notifications:process >> /var/log/vaultke/notifications.log 2>&1

# Send daily digest notifications at 8:00 AM
0 8 * * * cd /path/to/vaultke/backend && php artisan notifications:daily-digest >> /var/log/vaultke/notifications.log 2>&1

# Clean up old notifications and logs at 2:00 AM daily
0 2 * * * cd /path/to/vaultke/backend && php artisan notifications:cleanup >> /var/log/vaultke/notifications.log 2>&1

# Process batched notifications every 15 minutes
*/15 * * * * cd /path/to/vaultke/backend && php artisan notifications:process-batched >> /var/log/vaultke/notifications.log 2>&1

# Health check for notification system every 5 minutes
*/5 * * * * cd /path/to/vaultke/backend && php artisan notifications:health-check >> /var/log/vaultke/notifications.log 2>&1

# Backup notification data weekly (Sundays at 3:00 AM)
0 3 * * 0 cd /path/to/vaultke/backend && php artisan notifications:backup >> /var/log/vaultke/notifications.log 2>&1

# =====================================================
# Alternative: Using Laravel's Task Scheduler
# =====================================================

# If you prefer to use Laravel's built-in task scheduler,
# add this single cron job and define schedules in app/Console/Kernel.php:

# * * * * * cd /path/to/vaultke/backend && php artisan schedule:run >> /dev/null 2>&1

# Then in app/Console/Kernel.php:
/*
protected function schedule(Schedule $schedule)
{
    // Process notifications every minute
    $schedule->call(function () {
        $job = new ProcessNotificationsJob();
        $job->handle();
    })->everyMinute();

    // Send daily digests at 8 AM
    $schedule->call(function () {
        $job = new ProcessNotificationsJob();
        $job->sendDailyDigests();
    })->dailyAt('08:00');

    // Cleanup old data at 2 AM
    $schedule->call(function () {
        $job = new ProcessNotificationsJob();
        $job->cleanupOldNotifications();
    })->dailyAt('02:00');
}
*/

# =====================================================
# Log Rotation Configuration
# =====================================================

# Create log rotation config file: /etc/logrotate.d/vaultke-notifications
/*
/var/log/vaultke/notifications.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 www-data www-data
    postrotate
        systemctl reload rsyslog > /dev/null 2>&1 || true
    endscript
}
*/

# =====================================================
# Monitoring and Alerting
# =====================================================

# Monitor notification processing (every 10 minutes)
*/10 * * * * cd /path/to/vaultke/backend && php artisan notifications:monitor >> /var/log/vaultke/monitoring.log 2>&1

# Send alert if notification processing fails
*/30 * * * * cd /path/to/vaultke/backend && php artisan notifications:alert-on-failure >> /var/log/vaultke/alerts.log 2>&1

# =====================================================
# Performance Optimization
# =====================================================

# Optimize notification tables (weekly on Sundays at 4 AM)
0 4 * * 0 cd /path/to/vaultke/backend && php artisan notifications:optimize-tables >> /var/log/vaultke/maintenance.log 2>&1

# Update notification statistics cache (every hour)
0 * * * * cd /path/to/vaultke/backend && php artisan notifications:update-stats-cache >> /var/log/vaultke/cache.log 2>&1

# =====================================================
# Development Environment
# =====================================================

# For development, you might want less frequent processing:
# */5 * * * * cd /path/to/vaultke/backend && php artisan notifications:process >> /var/log/vaultke/notifications.log 2>&1

# =====================================================
# Production Environment Setup
# =====================================================

# 1. Create log directory:
# sudo mkdir -p /var/log/vaultke
# sudo chown www-data:www-data /var/log/vaultke
# sudo chmod 755 /var/log/vaultke

# 2. Create notification sound directory:
# sudo mkdir -p /var/www/vaultke/public/notification_sound
# sudo chown www-data:www-data /var/www/vaultke/public/notification_sound
# sudo chmod 755 /var/www/vaultke/public/notification_sound

# 3. Copy default notification sound:
# sudo cp /path/to/ring.mp3 /var/www/vaultke/public/notification_sound/
# sudo chown www-data:www-data /var/www/vaultke/public/notification_sound/ring.mp3
# sudo chmod 644 /var/www/vaultke/public/notification_sound/ring.mp3

# 4. Set up proper permissions:
# sudo chown -R www-data:www-data /var/www/vaultke/storage
# sudo chmod -R 775 /var/www/vaultke/storage

# 5. Install and configure supervisor for queue processing:
# sudo apt-get install supervisor
# sudo nano /etc/supervisor/conf.d/vaultke-notifications.conf

/*
[program:vaultke-notifications]
process_name=%(program_name)s_%(process_num)02d
command=php /var/www/vaultke/artisan queue:work --sleep=3 --tries=3 --timeout=60
directory=/var/www/vaultke
autostart=true
autorestart=true
user=www-data
numprocs=2
redirect_stderr=true
stdout_logfile=/var/log/vaultke/worker.log
stopwaitsecs=3600
*/

# 6. Start supervisor:
# sudo supervisorctl reread
# sudo supervisorctl update
# sudo supervisorctl start vaultke-notifications:*

# =====================================================
# Testing Cron Jobs
# =====================================================

# Test notification processing manually:
# cd /path/to/vaultke/backend && php artisan notifications:process

# Test with verbose output:
# cd /path/to/vaultke/backend && php artisan notifications:process --verbose

# Check cron job logs:
# tail -f /var/log/vaultke/notifications.log

# Check system cron logs:
# tail -f /var/log/syslog | grep CRON

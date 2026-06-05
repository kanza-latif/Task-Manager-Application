package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"taskmanager/internal/db/models"
	"taskmanager/internal/db/repository"

	"github.com/spf13/cobra"
)

// cgnat

var cgnatCmd = &cobra.Command{
	Use:   "cgnat",
	Short: "Query the cgnat_table",
}

var cgnatListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all CGNAT mappings",
	Example: `  tasks cgnat list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := repository.NewCGNATRepository(dbConn)
		entries, err := repo.List()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Println("No CGNAT entries found.")
			return nil
		}
		fmt.Printf("Total entries: %d\n\n", len(entries))
		printCGNATTable(entries)
		return nil
	},
}

var cgnatFindPrivateCmd = &cobra.Command{
	Use:     "find-private <ip>",
	Short:   "Find a CGNAT mapping by private IP",
	Example: `  tasks cgnat find-private 10.0.0.5`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := repository.NewCGNATRepository(dbConn)
		entries, err := repo.FindByPrivateIP(args[0])
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Printf("No CGNAT mapping found for private IP: %s\n", args[0])
			return nil
		}
		printCGNATTable(entries)
		return nil
	},
}

var cgnatFindPublicCmd = &cobra.Command{
	Use:     "find-public <ip>",
	Short:   "Find all CGNAT mappings sharing a public IP",
	Example: `  tasks cgnat find-public 203.0.113.10`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := repository.NewCGNATRepository(dbConn)
		entries, err := repo.FindByPublicIP(args[0])
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Printf("No CGNAT mappings found for public IP: %s\n", args[0])
			return nil
		}
		fmt.Printf("Found %d mapping(s) for public IP %s:\n\n", len(entries), args[0])
		printCGNATTable(entries)
		return nil
	},
}

// whitelist

var whitelistCmd = &cobra.Command{
	Use:   "whitelist",
	Short: "Query the whitelist_table",
}

var whitelistListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all whitelisted MSISDNs",
	Example: `  tasks whitelist list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := repository.NewWhitelistRepository(dbConn)
		count, err := repo.Count()
		if err != nil {
			return err
		}
		if count == 0 {
			fmt.Println("Whitelist is empty.")
			return nil
		}
		entries, err := repo.List()
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tMSISDN")
		fmt.Fprintln(w, "--\t------")
		for _, e := range entries {
			fmt.Fprintf(w, "%d\t%s\n", e.ID, e.MSISDN)
		}
		w.Flush()
		fmt.Printf("\nTotal whitelisted MSISDNs: %d\n", count)
		return nil
	},
}

var whitelistCheckCmd = &cobra.Command{
	Use:     "check <msisdn>",
	Short:   "Check if an MSISDN is whitelisted",
	Example: `  tasks whitelist check 923001234567`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		msisdn := args[0]
		repo := repository.NewWhitelistRepository(dbConn)
		found, err := repo.Check(msisdn)
		if err != nil {
			return err
		}
		if found {
			fmt.Printf("MSISDN %s is whitelisted.\n", msisdn)
		} else {
			fmt.Printf("MSISDN %s is NOT whitelisted.\n", msisdn)
		}
		return nil
	},
}

// user

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Query the user_table",
}

var userListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all users",
	Example: `  tasks user list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := repository.NewUserRepository(dbConn)
		users, err := repo.List()
		if err != nil {
			return err
		}
		if len(users) == 0 {
			fmt.Println("No users found.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tUSERNAME\tEMAIL\tTYPE\tSTATUS\tLAST LOGIN")
		fmt.Fprintln(w, "--\t--------\t-----\t----\t------\t----------")
		for _, u := range users {
			status := "active"
			if !u.Status {
				status = "inactive"
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
				u.ID, u.Username, u.Email, u.UserType, status,
				u.LastLogin.Format(time.RFC822))
		}
		w.Flush()
		return nil
	},
}

var userGetCmd = &cobra.Command{
	Use:     "get <username>",
	Short:   "Get details of a specific user",
	Example: `  tasks user get admin1`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := repository.NewUserRepository(dbConn)
		u, err := repo.GetByUsername(args[0])
		if err != nil {
			return err
		}
		status := "active"
		if !u.Status {
			status = "inactive"
		}
		fmt.Printf("\nUser #%d\n", u.ID)
		fmt.Printf("  Username  : %s\n", u.Username)
		fmt.Printf("  Email     : %s\n", u.Email)
		fmt.Printf("  Type      : %s\n", u.UserType)
		fmt.Printf("  Status    : %s\n", status)
		fmt.Printf("  Last Login: %s\n\n", u.LastLogin.Format(time.RFC1123))
		return nil
	},
}

var userTypeFilter string

var userListByTypeCmd = &cobra.Command{
	Use:     "filter",
	Short:   "List users by type",
	Example: `  tasks user filter --type admin`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if userTypeFilter == "" {
			return fmt.Errorf("provide --type: admin | viewer | whitelist")
		}
		repo := repository.NewUserRepository(dbConn)
		users, err := repo.ListByType(userTypeFilter)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			fmt.Printf("No users found with type: %s\n", userTypeFilter)
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tUSERNAME\tEMAIL\tSTATUS")
		fmt.Fprintln(w, "--\t--------\t-----\t------")
		for _, u := range users {
			status := "active"
			if !u.Status {
				status = "inactive"
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", u.ID, u.Username, u.Email, status)
		}
		w.Flush()
		return nil
	},
}

// session

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Query the session_table",
}

var sessionLimit int

var sessionListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List recent sessions",
	Example: `  tasks session list --limit 20`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := repository.NewSessionRepository(dbConn)
		sessions, err := repo.List(sessionLimit)
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			fmt.Println("No sessions found.")
			return nil
		}
		printSessionTable(sessions)
		return nil
	},
}

var sessionFindMSISDNCmd = &cobra.Command{
	Use:     "find <msisdn>",
	Short:   "Find all sessions for an MSISDN",
	Example: `  tasks session find 923001234567`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := repository.NewSessionRepository(dbConn)
		sessions, err := repo.FindByMSISDN(args[0])
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			fmt.Printf("No sessions found for MSISDN: %s\n", args[0])
			return nil
		}
		fmt.Printf("Found %d session(s) for MSISDN %s:\n\n", len(sessions), args[0])
		printSessionTable(sessions)
		return nil
	},
}

var (
	sessionFrom string
	sessionTo   string
)

var sessionTimeRangeCmd = &cobra.Command{
	Use:     "range",
	Short:   "Find sessions within a time range",
	Example: `  tasks session range --from "2026-05-19 00:00:00" --to "2026-05-19 23:59:59"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if sessionFrom == "" || sessionTo == "" {
			return fmt.Errorf("provide both --from and --to timestamps")
		}
		layout := "2006-01-02 15:04:05"
		from, err := time.Parse(layout, sessionFrom)
		if err != nil {
			return fmt.Errorf("invalid --from format, use: YYYY-MM-DD HH:MM:SS")
		}
		to, err := time.Parse(layout, sessionTo)
		if err != nil {
			return fmt.Errorf("invalid --to format, use: YYYY-MM-DD HH:MM:SS")
		}
		repo := repository.NewSessionRepository(dbConn)
		sessions, err := repo.FindByTimeRange(from, to)
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			fmt.Println("No sessions found in that time range.")
			return nil
		}
		fmt.Printf("Found %d session(s):\n\n", len(sessions))
		printSessionTable(sessions)
		return nil
	},
}

// alarm

var alarmCmd = &cobra.Command{
	Use:   "alarm",
	Short: "Query the alarm_table",
}

var alarmListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all alarms",
	Example: `  tasks alarm list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := repository.NewAlarmRepository(dbConn)
		alarms, err := repo.List()
		if err != nil {
			return err
		}
		if len(alarms) == 0 {
			fmt.Println("No alarms found.")
			return nil
		}
		printAlarmTable(alarms)
		return nil
	},
}

var alarmSeverityFilter string

var alarmSeverityCmd = &cobra.Command{
	Use:     "severity",
	Short:   "Filter alarms by severity",
	Example: `  tasks alarm severity --level critical`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if alarmSeverityFilter == "" {
			return fmt.Errorf("provide --level: critical | high | normal | low")
		}
		repo := repository.NewAlarmRepository(dbConn)
		alarms, err := repo.FindBySeverity(alarmSeverityFilter)
		if err != nil {
			return err
		}
		if len(alarms) == 0 {
			fmt.Printf("No alarms found with severity: %s\n", alarmSeverityFilter)
			return nil
		}
		printAlarmTable(alarms)
		return nil
	},
}

var alarmStatusFilter string

var alarmStatusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Filter alarms by status",
	Example: `  tasks alarm status --state new`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if alarmStatusFilter == "" {
			return fmt.Errorf("provide --state: new | resolved")
		}
		repo := repository.NewAlarmRepository(dbConn)
		alarms, err := repo.FindByStatus(alarmStatusFilter)
		if err != nil {
			return err
		}
		if len(alarms) == 0 {
			fmt.Printf("No alarms found with status: %s\n", alarmStatusFilter)
			return nil
		}
		printAlarmTable(alarms)
		return nil
	},
}

var alarmSiteFilter string

var alarmSiteCmd = &cobra.Command{
	Use:     "site",
	Short:   "Filter alarms by site",
	Example: `  tasks alarm site --name site1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if alarmSiteFilter == "" {
			return fmt.Errorf("provide --name for the site")
		}
		repo := repository.NewAlarmRepository(dbConn)
		alarms, err := repo.FindBySite(alarmSiteFilter)
		if err != nil {
			return err
		}
		if len(alarms) == 0 {
			fmt.Printf("No alarms found for site: %s\n", alarmSiteFilter)
			return nil
		}
		printAlarmTable(alarms)
		return nil
	},
}

func printCGNATTable(entries []*models.CGNAT) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tPRIVATE IP\tPUBLIC IP\tPORT RANGE")
	fmt.Fprintln(w, "--\t----------\t---------\t----------")
	for _, e := range entries {
		fmt.Fprintf(w, "%d\t%s\t%s\t%d – %d\n",
			e.ID, e.PrivateIP, e.PublicIP, e.StartPort, e.EndPort)
	}
	w.Flush()
}

func printSessionTable(sessions []*models.Session) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tMSISDN\tSITE\tPRIVATE IP\tPUBLIC IP\tPACKETS\tWL\tEND TIME")
	fmt.Fprintln(w, "--\t------\t----\t----------\t---------\t-------\t--\t--------")
	for _, s := range sessions {
		wl := "N"
		if s.WLStatus {
			wl = "Y"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			s.ID, s.MSISDN, s.Site, s.PrivateIP, s.PublicIP,
			s.Packets, wl, s.EndTime.Format(time.RFC822))
	}
	w.Flush()
}

func printAlarmTable(alarms []*models.Alarm) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tALARM ID\tSITE\tSEVERITY\tMODULE\tSTATUS\tTIME RAISED")
	fmt.Fprintln(w, "--\t--------\t----\t--------\t------\t------\t-----------")
	for _, a := range alarms {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			a.ID, a.AlarmID, a.Site, a.Severity,
			a.Module, a.Status, a.TimeRaised.Format(time.RFC822))
	}
	w.Flush()
}

// register all commands

func RegisterMDFCommands() {
	// cgnat
	cgnatCmd.AddCommand(cgnatListCmd, cgnatFindPrivateCmd, cgnatFindPublicCmd)

	// whitelist
	whitelistCmd.AddCommand(whitelistListCmd, whitelistCheckCmd)

	// user
	userListByTypeCmd.Flags().StringVarP(&userTypeFilter, "type", "t", "", "User type: admin | viewer | whitelist")
	userCmd.AddCommand(userListCmd, userGetCmd, userListByTypeCmd)

	// session
	sessionListCmd.Flags().IntVarP(&sessionLimit, "limit", "l", 50, "Number of sessions to fetch")
	sessionTimeRangeCmd.Flags().StringVar(&sessionFrom, "from", "", "Start time: YYYY-MM-DD HH:MM:SS")
	sessionTimeRangeCmd.Flags().StringVar(&sessionTo, "to", "", "End time: YYYY-MM-DD HH:MM:SS")
	sessionCmd.AddCommand(sessionListCmd, sessionFindMSISDNCmd, sessionTimeRangeCmd)

	// alarm
	alarmSeverityCmd.Flags().StringVarP(&alarmSeverityFilter, "level", "l", "", "Severity: critical | high | medium | low")
	alarmStatusCmd.Flags().StringVarP(&alarmStatusFilter, "state", "s", "", "Status: pending | in_progress | resolved")
	alarmSiteCmd.Flags().StringVarP(&alarmSiteFilter, "name", "n", "", "Site name")
	alarmCmd.AddCommand(alarmListCmd, alarmSeverityCmd, alarmStatusCmd, alarmSiteCmd)

	rootCmd.AddCommand(cgnatCmd, whitelistCmd, userCmd, sessionCmd, alarmCmd)
}

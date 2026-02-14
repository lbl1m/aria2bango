// Package firewall provides nftables management functionality
package firewall

import (
	"fmt"
	"net"
	"time"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
)

// NftablesManager manages nftables rules for blocking IPs
type NftablesManager struct {
	conn  *nftables.Conn
	table *nftables.Table
	setV4 *nftables.Set
	setV6 *nftables.Set
	chain *nftables.Chain
}

// NewNftablesManager creates a new nftables manager
func NewNftablesManager(tableName string) (*NftablesManager, error) {
	conn, err := nftables.New()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nftables: %w", err)
	}

	mgr := &NftablesManager{
		conn: conn,
	}

	// Initialize table and sets
	if err := mgr.init(tableName); err != nil {
		conn.CloseLasting()
		return nil, err
	}

	return mgr, nil
}

// init initializes the nftables table, sets, and chain
func (m *NftablesManager) init(tableName string) error {
	// Create or get the table
	m.table = &nftables.Table{
		Name:   tableName,
		Family: nftables.TableFamilyINet,
	}

	// Add the table
	m.conn.AddTable(m.table)

	// Create IPv4 set with timeout support
	m.setV4 = &nftables.Set{
		Name:       "blocked_v4",
		Table:      m.table,
		KeyType:    nftables.TypeIPAddr,
		HasTimeout: true,
	}

	if err := m.conn.AddSet(m.setV4, nil); err != nil {
		// Set might already exist, try to find it
		sets, err := m.conn.GetSets(m.table)
		if err != nil {
			return fmt.Errorf("failed to get sets: %w", err)
		}
		for _, s := range sets {
			if s.Name == "blocked_v4" {
				m.setV4 = s
				break
			}
		}
	}

	// Create IPv6 set with timeout support
	m.setV6 = &nftables.Set{
		Name:       "blocked_v6",
		Table:      m.table,
		KeyType:    nftables.TypeIP6Addr,
		HasTimeout: true,
	}

	if err := m.conn.AddSet(m.setV6, nil); err != nil {
		// Set might already exist, try to find it
		sets, err := m.conn.GetSets(m.table)
		if err != nil {
			return fmt.Errorf("failed to get sets: %w", err)
		}
		for _, s := range sets {
			if s.Name == "blocked_v6" {
				m.setV6 = s
				break
			}
		}
	}

	// Create the output chain - we block outgoing packets to leechers
	// This prevents them from downloading from us, but we can still download from them
	m.chain = &nftables.Chain{
		Name:     "output",
		Table:    m.table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityFilter,
	}
	m.conn.AddChain(m.chain)

	// Add rule to drop outgoing packets to blocked IPv4 addresses
	// This blocks us from sending data to leechers (they can't download from us)
	// But we can still receive data from them (we can download from them)
	m.conn.AddRule(&nftables.Rule{
		Table: m.table,
		Chain: m.chain,
		Exprs: []expr.Any{
			// Load destination IP (we're blocking outgoing packets)
			&expr.Payload{
				OperationType: expr.PayloadLoad,
				Len:           4,
				Offset:        16, // Destination IP offset in IPv4 header
				DestRegister:  1,
				Base:          expr.PayloadBaseNetworkHeader,
			},
			// Check if destination IP is in set
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        m.setV4.Name,
				SetID:          m.setV4.ID,
			},
			// Drop the packet
			&expr.Verdict{
				Kind: expr.VerdictDrop,
			},
		},
	})

	// Add rule to drop outgoing packets to blocked IPv6 addresses
	m.conn.AddRule(&nftables.Rule{
		Table: m.table,
		Chain: m.chain,
		Exprs: []expr.Any{
			// Load destination IPv6 (we're blocking outgoing packets)
			&expr.Payload{
				OperationType: expr.PayloadLoad,
				Len:           16,
				Offset:        24, // Destination IPv6 offset in IPv6 header
				DestRegister:  1,
				Base:          expr.PayloadBaseNetworkHeader,
			},
			// Check if destination IP is in set
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        m.setV6.Name,
				SetID:          m.setV6.ID,
			},
			// Drop the packet
			&expr.Verdict{
				Kind: expr.VerdictDrop,
			},
		},
	})

	if err := m.conn.Flush(); err != nil {
		return fmt.Errorf("failed to flush nftables: %w", err)
	}

	return nil
}

// BlockIP adds an IP to the blocked set with the specified duration
func (m *NftablesManager) BlockIP(ipStr string, duration time.Duration) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", ipStr)
	}

	if ip.To4() != nil {
		// IPv4
		elements := []nftables.SetElement{
			{
				Key:     ip.To4(),
				Timeout: duration,
			},
		}
		if err := m.conn.SetAddElements(m.setV4, elements); err != nil {
			return fmt.Errorf("failed to add IPv4 to set: %w", err)
		}
	} else {
		// IPv6
		elements := []nftables.SetElement{
			{
				Key:     ip.To16(),
				Timeout: duration,
			},
		}
		if err := m.conn.SetAddElements(m.setV6, elements); err != nil {
			return fmt.Errorf("failed to add IPv6 to set: %w", err)
		}
	}

	return m.conn.Flush()
}

// UnblockIP removes an IP from the blocked set
func (m *NftablesManager) UnblockIP(ipStr string) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", ipStr)
	}

	if ip.To4() != nil {
		// IPv4
		elements := []nftables.SetElement{
			{Key: ip.To4()},
		}
		if err := m.conn.SetDeleteElements(m.setV4, elements); err != nil {
			return fmt.Errorf("failed to remove IPv4 from set: %w", err)
		}
	} else {
		// IPv6
		elements := []nftables.SetElement{
			{Key: ip.To16()},
		}
		if err := m.conn.SetDeleteElements(m.setV6, elements); err != nil {
			return fmt.Errorf("failed to remove IPv6 from set: %w", err)
		}
	}

	return m.conn.Flush()
}

// Clear removes all blocked IPs
func (m *NftablesManager) Clear() error {
	// Flush all elements from sets
	m.conn.FlushSet(m.setV4)
	m.conn.FlushSet(m.setV6)
	return m.conn.Flush()
}

// Destroy removes the entire table
func (m *NftablesManager) Destroy() error {
	m.conn.DelTable(m.table)
	return m.conn.Flush()
}

// ListBlocked returns all currently blocked IPs
func (m *NftablesManager) ListBlocked() ([]string, error) {
	var blockedIPs []string

	// Get IPv4 elements
	v4Elements, err := m.conn.GetSetElements(m.setV4)
	if err != nil {
		return nil, fmt.Errorf("failed to get IPv4 set elements: %w", err)
	}
	for _, elem := range v4Elements {
		blockedIPs = append(blockedIPs, net.IP(elem.Key).String())
	}

	// Get IPv6 elements
	v6Elements, err := m.conn.GetSetElements(m.setV6)
	if err != nil {
		return nil, fmt.Errorf("failed to get IPv6 set elements: %w", err)
	}
	for _, elem := range v6Elements {
		blockedIPs = append(blockedIPs, net.IP(elem.Key).String())
	}

	return blockedIPs, nil
}

// Close closes the nftables connection
func (m *NftablesManager) Close() error {
	m.conn.CloseLasting()
	return nil
}

// Cleanup cleans up any leftover nftables rules from previous runs
func Cleanup(tableName string) error {
	conn, err := nftables.New()
	if err != nil {
		return fmt.Errorf("failed to connect to nftables: %w", err)
	}
	defer conn.CloseLasting()

	table := &nftables.Table{
		Name:   tableName,
		Family: nftables.TableFamilyINet,
	}

	conn.DelTable(table)
	return conn.Flush()
}

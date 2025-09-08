
```vhdl
-- D-FlipFlop Implementation
library IEEE;
use IEEE.STD_LOGIC_1164.ALL;

entity D_FlipFlop is
    Port ( 
        D     : in  STD_LOGIC;
        CLK   : in  STD_LOGIC;
        RESET : in  STD_LOGIC;
        Q     : out STD_LOGIC;
        Q_N   : out STD_LOGIC
    );
end D_FlipFlop;

architecture Behavioral of D_FlipFlop is
    signal Q_internal : STD_LOGIC := '0';
begin
    process(CLK, RESET)
    begin
        if RESET = '1' then
            Q_internal <= '0';
        elsif rising_edge(CLK) then
            Q_internal <= D;
        end if;
    end process;
    
    Q <= Q_internal;
    Q_N <= not Q_internal;
end Behavioral;

-- JK-FlipFlop Implementation
library IEEE;
use IEEE.STD_LOGIC_1164.ALL;

entity JK_FlipFlop is
    Port ( 
        J     : in  STD_LOGIC;
        K     : in  STD_LOGIC;
        CLK   : in  STD_LOGIC;
        RESET : in  STD_LOGIC;
        Q     : out STD_LOGIC;
        Q_N   : out STD_LOGIC
    );
end JK_FlipFlop;

architecture Behavioral of JK_FlipFlop is
    signal Q_internal : STD_LOGIC := '0';
begin
    process(CLK, RESET)
    begin
        if RESET = '1' then
            Q_internal <= '0';
        elsif rising_edge(CLK) then
            case (J & K) is
                when "00" => Q_internal <= Q_internal;  -- Hold
                when "01" => Q_internal <= '0';         -- Reset
                when "10" => Q_internal <= '1';         -- Set
                when "11" => Q_internal <= not Q_internal; -- Toggle
                when others => Q_internal <= Q_internal;
            end case;
        end if;
    end process;
    
    Q <= Q_internal;
    Q_N <= not Q_internal;
end Behavioral;

-- Test Bench for D-FlipFlop
library IEEE;
use IEEE.STD_LOGIC_1164.ALL;

entity D_FlipFlop_TB is
end D_FlipFlop_TB;

architecture Behavioral of D_FlipFlop_TB is
    -- Component declaration
    component D_FlipFlop
        Port ( 
            D     : in  STD_LOGIC;
            CLK   : in  STD_LOGIC;
            RESET : in  STD_LOGIC;
            Q     : out STD_LOGIC;
            Q_N   : out STD_LOGIC
        );
    end component;
    
    -- Test signals
    signal D_tb     : STD_LOGIC := '0';
    signal CLK_tb   : STD_LOGIC := '0';
    signal RESET_tb : STD_LOGIC := '0';
    signal Q_tb     : STD_LOGIC;
    signal Q_N_tb   : STD_LOGIC;
    
    -- Clock period definition
    constant CLK_period : time := 20 ns;
    
begin
    -- Instantiate the Unit Under Test (UUT)
    uut: D_FlipFlop PORT MAP (
        D => D_tb,
        CLK => CLK_tb,
        RESET => RESET_tb,
        Q => Q_tb,
        Q_N => Q_N_tb
    );
    
    -- Clock process
    CLK_process: process
    begin
        CLK_tb <= '0';
        wait for CLK_period/2;
        CLK_tb <= '1';
        wait for CLK_period/2;
    end process;
    
    -- Stimulus process
    stim_proc: process
    begin
        -- Test 1: Reset functionality
        RESET_tb <= '1';
        D_tb <= '1';
        wait for CLK_period;
        assert Q_tb = '0' report "Reset test failed" severity error;
        
        RESET_tb <= '0';
        wait for CLK_period;
        
        -- Test 2: D input following
        D_tb <= '1';
        wait for CLK_period;
        assert Q_tb = '1' report "D=1 test failed" severity error;
        
        D_tb <= '0';
        wait for CLK_period;
        assert Q_tb = '0' report "D=0 test failed" severity error;
        
        D_tb <= '1';
        wait for CLK_period;
        assert Q_tb = '1' report "D=1 second test failed" severity error;
        
        wait for CLK_period * 2;
        
        report "D-FlipFlop test completed";
        wait;
    end process;
end Behavioral;

-- Test Bench for JK-FlipFlop
library IEEE;
use IEEE.STD_LOGIC_1164.ALL;

entity JK_FlipFlop_TB is
end JK_FlipFlop_TB;

architecture Behavioral of JK_FlipFlop_TB is
    -- Component declaration
    component JK_FlipFlop
        Port ( 
            J     : in  STD_LOGIC;
            K     : in  STD_LOGIC;
            CLK   : in  STD_LOGIC;
            RESET : in  STD_LOGIC;
            Q     : out STD_LOGIC;
            Q_N   : out STD_LOGIC
        );
    end component;
    
    -- Test signals
    signal J_tb     : STD_LOGIC := '0';
    signal K_tb     : STD_LOGIC := '0';
    signal CLK_tb   : STD_LOGIC := '0';
    signal RESET_tb : STD_LOGIC := '0';
    signal Q_tb     : STD_LOGIC;
    signal Q_N_tb   : STD_LOGIC;
    
    -- Clock period definition
    constant CLK_period : time := 20 ns;
    
begin
    -- Instantiate the Unit Under Test (UUT)
    uut: JK_FlipFlop PORT MAP (
        J => J_tb,
        K => K_tb,
        CLK => CLK_tb,
        RESET => RESET_tb,
        Q => Q_tb,
        Q_N => Q_N_tb
    );
    
    -- Clock process
    CLK_process: process
    begin
        CLK_tb <= '0';
        wait for CLK_period/2;
        CLK_tb <= '1';
        wait for CLK_period/2;
    end process;
    
    -- Stimulus process
    stim_proc: process
    begin
        -- Test 1: Reset and basic JK operations
        RESET_tb <= '1';
        wait for CLK_period;
        assert Q_tb = '0' report "Reset test failed" severity error;
        
        RESET_tb <= '0';
        wait for CLK_period/2;
        
        -- Test Set (J=1, K=0)
        J_tb <= '1';
        K_tb <= '0';
        wait for CLK_period;
        assert Q_tb = '1' report "Set test failed" severity error;
        
        -- Test Hold (J=0, K=0)
        J_tb <= '0';
        K_tb <= '0';
        wait for CLK_period;
        assert Q_tb = '1' report "Hold test failed" severity error;
        
        -- Test Reset (J=0, K=1)
        J_tb <= '0';
        K_tb <= '1';
        wait for CLK_period;
        assert Q_tb = '0' report "Reset test failed" severity error;
        
        -- Test 2: Toggle functionality (J=1, K=1)
        J_tb <= '1';
        K_tb <= '1';
        wait for CLK_period;
        assert Q_tb = '1' report "First toggle test failed" severity error;
        
        -- Continue toggling
        wait for CLK_period;
        assert Q_tb = '0' report "Second toggle test failed" severity error;
        
        wait for CLK_period;
        assert Q_tb = '1' report "Third toggle test failed" severity error;
        
        wait for CLK_period * 2;
        
        report "JK-FlipFlop test completed";
        wait;
    end process;
end Behavioral;

```

library ieee;
use ieee.std_logic_1164.all;
use ieee.numeric_std.all;

entity PC is
  port(
    clk           : in std_logic;
    current_instr : in std_logic_vector(2 downto 0);   -- current instruction
    next_instr    : out std_logic_vector(2 downto 0)   -- next instruction
  );
end PC;


architecture behavioral of PC is

  signal next_signal : std_logic_vector(2 downto 0);

begin
  process(clk)
  begin
    if (current_instr = "UUU") then
      next_signal <= "000";
    end if;
    if falling_edge(clk) then
      next_signal <= std_logic_vector(unsigned(current_instr) + to_unsigned(1, 3));
    end if;
  end process;

  next_instr <= next_signal;

end behavioral;

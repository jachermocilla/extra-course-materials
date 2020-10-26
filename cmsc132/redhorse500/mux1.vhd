library ieee;
use ieee.std_logic_1164.all;

entity mux1 is
  port(
    sel : in std_logic;                      -- select data
    a   : in std_logic_vector(7 downto 0);   -- default data
    b   : in std_logic_vector(7 downto 0);   -- data from instruction
    y   : out std_logic_vector(7 downto 0)   -- data out
  );
end mux1;


architecture dataflow of mux1 is

begin
  process(sel, a, b)
  begin
    if (sel = '0') then
      y <= a;
    else
      y <= b;
    end if;
  end process;

end dataflow;

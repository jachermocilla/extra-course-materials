library ieee;
use ieee.std_logic_1164.all;

entity mux0 is
  port(
    sel : in std_logic;                      -- select destination address
    a   : in std_logic_vector(1 downto 0);   -- source register address
    b   : in std_logic_vector(1 downto 0);   -- default destination address
    y   : out std_logic_vector(1 downto 0)   -- destination address
  );
end mux0;


architecture dataflow of mux0 is

begin
  process(sel, a, b)
  begin
    if (sel = '1') then
      y <= a;
    else
      y <= b;
    end if;
  end process;

end dataflow;
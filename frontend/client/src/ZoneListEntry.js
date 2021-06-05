

var ZoneListEntry = ({zone}) => {
  console.log(zone);
  var zoneName = zone.name;
  return (
    <div className = 'zone-list-entry'>
     {zoneName}
    </div>
  )



}

export default ZoneListEntry;
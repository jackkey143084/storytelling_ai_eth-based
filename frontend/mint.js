document.getElementById('mint').onclick = async () => {
  const to = document.getElementById('to').value;
  const cid = document.getElementById('cid').value;
  const checkpoint = document.getElementById('checkpoint').value;
  const res = await fetch('http://localhost:8081/mint', {
    method: 'POST',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify({to_address: to, ipfs_cid: cid, checkpoint})
  });
  const j = await res.json();
  document.getElementById('out').textContent = JSON.stringify(j, null, 2);
};

